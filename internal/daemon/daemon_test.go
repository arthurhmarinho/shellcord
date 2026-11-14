package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/arthurhmarinho/shellcord/internal/discord/discordtest"
	"github.com/arthurhmarinho/shellcord/internal/event"
	"github.com/arthurhmarinho/shellcord/internal/proc"
	"github.com/arthurhmarinho/shellcord/internal/state"
)

func TestRestartWhenExecutableChanges(t *testing.T) {
	dir := t.TempDir()
	exe := dir + "/shellcord"
	if err := os.WriteFile(exe, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Config{ClientID: "123", Executable: exe, ConfigPath: dir + "/c.toml", Dir: dir,
		DiscordDirs: []string{t.TempDir()}, Check: 30 * time.Millisecond, Log: Discard}
	done := make(chan error, 1)
	go func() { done <- Run(context.Background(), cfg) }()
	waitSocket(t, dir)

	select {
	case err := <-done:
		t.Fatalf("saiu sem o executável mudar: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := os.WriteFile(exe+".novo", []byte("versão 2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(exe+".novo", exe); err != nil { // como o go install faz
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrRestart) {
			t.Fatalf("quero ErrRestart, veio %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("não percebeu o executável novo")
	}
	if _, err := os.Stat(SocketPath(dir)); !os.IsNotExist(err) {
		t.Fatal("antes de reexecutar, o socket deveria ter sido removido")
	}
}

func TestConfigFailClosedAndReload(t *testing.T) {
	srv := discordtest.New(t, "123")
	dir := t.TempDir()
	confPath := dir + "/config.toml"
	if err := os.WriteFile(confPath, []byte(`ignore_dir = ["/h"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{ClientID: "123", ConfigPath: confPath, Dir: dir, DiscordDirs: []string{srv.Dir}, Home: "/h",
		Window: 100 * time.Millisecond, Check: 50 * time.Millisecond, Log: Discard}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Run(ctx, cfg)
	waitSocket(t, dir)

	me, _ := proc.Read(os.Getpid())
	ev := func(cwd string) event.Event {
		return event.Event{V: 1, Src: event.SourceFish, PID: me.PID, Start: me.Start, Kind: event.KindExec, Prog: "nvim", Cwd: cwd}
	}
	send(t, dir, ev("/h/publico"))
	srv.NoActivity(t, 400*time.Millisecond) // config inválida: falha fechado

	// Corrige a config (tamanho diferente garante que a mudança é notada).
	if err := os.WriteFile(confPath, []byte(`ignore_dirs = ["~/segredo"]`+"\nallow = [\"nvim\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := srv.NextActivity(t, 2*time.Second); !strings.Contains(got, "Rodando nvim") {
		t.Fatalf("com a config corrigida o card deveria aparecer: %s", got)
	}
	send(t, dir, ev("/h/segredo/projeto"))
	if got := srv.NextActivity(t, 2*time.Second); got != "null" {
		t.Fatalf("na pasta ignorada a fonte some: %s", got)
	}

	// Uma config inválida depois de uma válida mantém a anterior.
	if err := os.WriteFile(confPath, []byte(`isso não é toml, e é bem mais comprido`), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond) // várias checagens: a config inválida já foi lida
	send(t, dir, ev("/h/publico"))
	if got := srv.NextActivity(t, 2*time.Second); !strings.Contains(got, "Rodando nvim") {
		t.Fatalf("config inválida numa releitura mantém a anterior: %s", got)
	}
}

func TestRender(t *testing.T) {
	since := time.Unix(1000, 0)
	allow := map[string]bool{"nvim": true}
	cases := []struct {
		sum            state.Summary
		details, state string
	}{
		{state.Summary{Level: state.LevelClaudeBusy, Count: 1, Cwd: "/h/proj", Since: since}, "Claude Code trabalhando", "em proj"},
		{state.Summary{Level: state.LevelClaudeBusy, Count: 2, Cwd: "/h/proj", Since: since}, "Claude Code trabalhando · 2 sessões", "em proj"},
		{state.Summary{Level: state.LevelClaudeBusy, Count: 1, Tool: "Edit", Cwd: "/h/proj", Since: since}, "Claude Code editando arquivos", "em proj"},
		{state.Summary{Level: state.LevelClaudeBusy, Count: 1, Tool: "Bash", Since: since}, "Claude Code rodando comandos", ""},
		{state.Summary{Level: state.LevelClaudeBusy, Count: 1, Tool: "mcp__gmail__send", Since: since}, "Claude Code trabalhando", ""},
		{state.Summary{Level: state.LevelClaudeIdle, Count: 2, Cwd: "/h/proj", Since: since}, "Claude Code esperando você · 2 sessões", "em proj"},
		{state.Summary{Level: state.LevelShellExec, Count: 1, Prog: "nvim", Cwd: "/h", Since: since}, "Rodando nvim", "em ~"},
		{state.Summary{Level: state.LevelShellExec, Count: 1, Prog: "segredo", Cwd: "/h/x/", Since: since}, "No terminal", "em x"},
		{state.Summary{Level: state.LevelShellIdle, Count: 3, Since: since}, "No terminal · 3 terminais", ""},
	}
	for _, c := range cases {
		a := render(c.sum, "/h", allow)
		if a == nil || a.Details != c.details || a.State != c.state || a.Timestamps.Start != 1000 {
			t.Errorf("render(%+v) = %+v, quero %q / %q", c.sum, a, c.details, c.state)
		}
	}
	if a := render(cases[0].sum, "/h", allow); a.Assets == nil || a.Assets.LargeImage != cardImage {
		t.Errorf("o card deveria levar a imagem %q: %+v", cardImage, a.Assets)
	}
	if render(state.Summary{}, "/h", allow) != nil {
		t.Error("nível nenhum deveria limpar a presença")
	}
	if a := render(state.Summary{Level: state.LevelShellIdle, Cwd: "/" + strings.Repeat("a", 300)}, "", nil); len([]rune(a.State)) != maxField {
		t.Errorf("state deveria ser truncado em %d, tem %d", maxField, len([]rune(a.State)))
	}
}

func send(t *testing.T, dir string, e event.Event) {
	t.Helper()
	b, err := event.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: SocketPath(dir), Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write(b); err != nil {
		t.Fatal(err)
	}
}

func waitSocket(t *testing.T, dir string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if _, err := os.Stat(SocketPath(dir)); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("o daemon não criou o socket")
}

// TestEndToEnd sobe o daemon com um Discord falso e uma janela curta e confere
// o caminho evento → card, a coalescência, o lock e a reconexão.
func TestEndToEnd(t *testing.T) {
	srv := discordtest.New(t, "123")
	dir := t.TempDir()
	cfg := Config{ClientID: "123", ConfigPath: dir + "/sem-config.toml", Dir: dir, DiscordDirs: []string{srv.Dir}, Home: "/h",
		Window: 300 * time.Millisecond, Check: 50 * time.Millisecond, Log: Discard}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg) }()
	waitSocket(t, dir)

	if err := Run(context.Background(), cfg); !errors.Is(err, ErrRunning) {
		t.Fatalf("segundo daemon deveria dar ErrRunning, deu %v", err)
	}

	me, err := proc.Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	fish := func(kind event.Kind, prog string) event.Event {
		return event.Event{V: 1, Src: event.SourceFish, PID: me.PID, Start: me.Start, Kind: kind, Prog: prog, Cwd: "/h/projeto"}
	}

	send(t, dir, fish(event.KindExec, "nvim"))
	if got := srv.NextActivity(t, 2*time.Second); !strings.Contains(got, "Rodando nvim") || !strings.Contains(got, "em projeto") {
		t.Fatalf("primeiro card: %s", got)
	}

	// Dentro da janela: três eventos viram um envio só, com o estado final.
	send(t, dir, fish(event.KindExec, "git"))
	send(t, dir, fish(event.KindExec, "make"))
	send(t, dir, fish(event.KindExec, "go"))
	if got := srv.NextActivity(t, 2*time.Second); !strings.Contains(got, "Rodando go") {
		t.Fatalf("depois da janela deveria vir só o estado final: %s", got)
	}
	srv.NoActivity(t, 400*time.Millisecond)

	// O Discord fecha e reabre: o daemon reconecta e reenvia o estado atual.
	srv.Disconnect()
	if got := srv.NextActivity(t, 20*time.Second); !strings.Contains(got, "Rodando go") {
		t.Fatalf("depois de reconectar: %s", got)
	}

	send(t, dir, fish(event.KindEnd, ""))
	if got := srv.NextActivity(t, 2*time.Second); got != "null" {
		t.Fatalf("sem fontes a presença deveria ser limpa: %s", got)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(SocketPath(dir)); !os.IsNotExist(err) {
		t.Fatal("o socket deveria ser removido ao sair")
	}
}
