// Package daemon é o processo residente: recebe eventos pelo socket local,
// mantém o estado das fontes e publica o card no Discord respeitando a janela
// de atualização (ADR-0002, ADR-0003).
package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"

	"github.com/arthurhmarinho/shellcord/internal/discord"
	"github.com/arthurhmarinho/shellcord/internal/event"
	"github.com/arthurhmarinho/shellcord/internal/proc"
	"github.com/arthurhmarinho/shellcord/internal/state"
)

// ErrRunning indica que outro daemon já segura o lock.
var ErrRunning = errors.New("daemon já está rodando")

// Config configura o daemon. Campos zerados recebem os padrões.
type Config struct {
	ClientID    string
	Dir         string        // padrão: RuntimeDir()
	DiscordDirs []string      // padrão: discord.DefaultDirs()
	Home        string        // padrão: os.UserHomeDir()
	Window      time.Duration // intervalo mínimo entre SET_ACTIVITY; padrão 15 s
	Check       time.Duration // intervalo da checagem periódica; padrão 15 s
	IdleTimeout time.Duration // shell parado some depois disso; padrão 10 min
	SourceTTL   time.Duration // fonte sem PID expira depois disso; padrão 10 min
	Log         *log.Logger   // padrão: stderr
}

func (c *Config) setDefaults() {
	if c.Dir == "" {
		c.Dir = RuntimeDir()
	}
	if c.DiscordDirs == nil {
		c.DiscordDirs = discord.DefaultDirs()
	}
	if c.Home == "" {
		c.Home, _ = os.UserHomeDir()
	}
	if c.Window == 0 {
		c.Window = 15 * time.Second
	}
	if c.Check == 0 {
		c.Check = 15 * time.Second
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 10 * time.Minute
	}
	if c.SourceTTL == 0 {
		c.SourceTTL = 10 * time.Minute
	}
	if c.Log == nil {
		c.Log = log.New(os.Stderr, "shellcord: ", log.LstdFlags)
	}
}

// RuntimeDir é o diretório do shellcord em $XDG_RUNTIME_DIR (ADR-0009).
func RuntimeDir() string {
	rt := os.Getenv("XDG_RUNTIME_DIR")
	if rt == "" {
		rt = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return filepath.Join(rt, "shellcord")
}

// SocketPath é o socket de eventos dentro de dir.
func SocketPath(dir string) string { return filepath.Join(dir, "events.sock") }

// Run executa o daemon até ctx ser cancelado.
func Run(ctx context.Context, cfg Config) error {
	cfg.setDefaults()
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(cfg.Dir, 0o700); err != nil {
		return err
	}

	// O lock impede dois daemons (sway e autoarranque ao mesmo tempo, ADR-0006).
	// O kernel solta o flock sozinho se o processo morrer.
	lock, err := os.OpenFile(filepath.Join(cfg.Dir, "daemon.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return ErrRunning
		}
		return err
	}

	sock := SocketPath(cfg.Dir)
	os.Remove(sock) // sobra de um daemon morto; com o lock, não é de um vivo
	pc, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: sock, Net: "unixgram"})
	if err != nil {
		return err
	}
	defer os.Remove(sock)
	defer pc.Close()

	events := make(chan event.Event, 64)
	readErr := make(chan error, 1)
	go readEvents(ctx, pc, events, readErr, cfg.Log)

	l := &loop{
		cfg:   cfg,
		store: state.New(state.Options{TTL: cfg.SourceTTL, IdleTimeout: cfg.IdleTimeout, Alive: proc.Alive}),
	}
	cfg.Log.Printf("ouvindo em %s", sock)
	return l.run(ctx, events, readErr)
}

func readEvents(ctx context.Context, pc *net.UnixConn, out chan<- event.Event, errc chan<- error, lg *log.Logger) {
	buf := make([]byte, event.MaxSize+1)
	for {
		n, _, err := pc.ReadFromUnix(buf)
		if err != nil {
			if ctx.Err() == nil && !errors.Is(err, net.ErrClosed) {
				errc <- err
			}
			return
		}
		e, err := event.Decode(buf[:n])
		if err != nil {
			lg.Printf("evento descartado: %v", err)
			continue
		}
		select {
		case out <- e:
		case <-ctx.Done():
			return
		}
	}
}

const (
	backoffMin = 15 * time.Second
	backoffMax = 2 * time.Minute
)

type loop struct {
	cfg   Config
	store *state.Store

	client     *discord.Client
	clientDone <-chan struct{} // nil enquanto desconectado

	sent      *discord.Activity // o que o Discord está mostrando
	sentValid bool              // false força reenviar (ex.: depois de reconectar)
	lastSend  time.Time

	nextDial time.Time
	backoff  time.Duration

	timer      *time.Timer // dispara quando a janela libera
	timerArmed bool
}

func (l *loop) run(ctx context.Context, events <-chan event.Event, readErr <-chan error) error {
	ticker := time.NewTicker(l.cfg.Check)
	defer ticker.Stop()
	l.timer = time.NewTimer(time.Hour)
	l.timer.Stop()
	l.backoff = backoffMin
	defer func() {
		if l.client != nil {
			l.client.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-readErr:
			return fmt.Errorf("socket de eventos: %w", err)
		case e := <-events:
			l.store.Apply(e, time.Now())
			l.flush(time.Now())
		case <-ticker.C:
			l.flush(time.Now())
		case <-l.timer.C:
			l.timerArmed = false
			l.flush(time.Now())
		case <-l.clientDone:
			l.cfg.Log.Printf("conexão com o Discord caiu: %v", l.client.Err())
			l.disconnect()
			l.flush(time.Now())
		}
	}
}

// flush calcula o card desejado e o envia se mudou, respeitando a janela. Os
// estados intermediários dentro da janela são descartados: só o mais recente
// é enviado quando ela libera (ADR-0003).
func (l *loop) flush(now time.Time) {
	l.store.Prune(now)
	want := render(l.store.Summary(now), l.cfg.Home, defaultAllow)
	if l.sentValid && reflect.DeepEqual(want, l.sent) {
		return
	}
	if l.client == nil && want == nil {
		// Desconectado não mostra nada: não há o que limpar.
		l.sent, l.sentValid = nil, true
		return
	}
	if !l.connect(now) {
		return // o ticker tenta de novo
	}
	if wait := l.lastSend.Add(l.cfg.Window).Sub(now); wait > 0 {
		if !l.timerArmed {
			l.timer.Reset(wait)
			l.timerArmed = true
		}
		return
	}
	if err := l.client.SetActivity(want); err != nil {
		l.cfg.Log.Printf("SET_ACTIVITY falhou: %v", err)
		l.disconnect()
		return
	}
	l.sent, l.sentValid, l.lastSend = want, true, now
	if want == nil {
		l.cfg.Log.Print("card: limpo")
	} else {
		l.cfg.Log.Printf("card: %q / %q", want.Details, want.State)
	}
}

// connect garante uma conexão, com backoff entre tentativas falhas.
func (l *loop) connect(now time.Time) bool {
	if l.client != nil {
		return true
	}
	if now.Before(l.nextDial) {
		return false
	}
	c, err := discord.Dial(l.cfg.ClientID, l.cfg.DiscordDirs)
	if err != nil {
		l.cfg.Log.Printf("%v (nova tentativa em %s)", err, l.backoff)
		l.nextDial = now.Add(l.backoff)
		l.backoff = min(l.backoff*2, backoffMax)
		return false
	}
	l.cfg.Log.Print("conectado ao Discord")
	l.client, l.clientDone = c, c.Done()
	l.sentValid = false
	l.backoff, l.nextDial = backoffMin, time.Time{}
	return true
}

func (l *loop) disconnect() {
	if l.client != nil {
		l.client.Close()
	}
	l.client, l.clientDone = nil, nil
	l.sent, l.sentValid = nil, false
}

// Discard é um logger que descarta tudo, útil em testes.
var Discard = log.New(io.Discard, "", 0)
