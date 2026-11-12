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

func TestRender(t *testing.T) {
	since := time.Unix(1000, 0)
	allow := map[string]bool{"nvim": true}
	cases := []struct {
		sum            state.Summary
		details, state string
	}{
		{state.Summary{Level: state.LevelClaudeBusy, Count: 1, Cwd: "/h/proj", Since: since}, "Claude Code trabalhando", "em proj"},
		{state.Summary{Level: state.LevelClaudeBusy, Count: 2, Cwd: "/h/proj", Since: since}, "Claude Code trabalhando · 2 sessões", "em proj"},
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
	cfg := Config{ClientID: "123", Dir: dir, DiscordDirs: []string{srv.Dir}, Home: "/h",
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
