package state

import (
	"testing"
	"time"

	"github.com/arthurhmarinho/shellcord/internal/event"
)

var t0 = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func at(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }

type fakeProcs map[int]bool

func (f fakeProcs) alive(pid int, _ uint64) bool { return f[pid] }

func newStore(procs fakeProcs) *Store {
	return New(Options{TTL: 10 * time.Minute, IdleTimeout: 10 * time.Minute, Alive: procs.alive})
}

func fish(pid int, kind event.Kind, prog, cwd string) event.Event {
	return event.Event{V: 1, Src: event.SourceFish, PID: pid, Kind: kind, Prog: prog, Cwd: cwd}
}

func claudeTool(id, tool string) event.Event {
	return event.Event{V: 1, Src: event.SourceClaude, ID: id, Kind: event.KindBusy, Tool: tool, Cwd: "/p"}
}

func claude(id string, pid int, kind event.Kind, cwd string) event.Event {
	return event.Event{V: 1, Src: event.SourceClaude, ID: id, PID: pid, Kind: kind, Cwd: cwd}
}

func TestEmpty(t *testing.T) {
	if got := newStore(nil).Summary(t0); got.Level != LevelNone {
		t.Fatalf("store vazio deveria limpar a presença: %+v", got)
	}
}

func TestPriority(t *testing.T) {
	s := newStore(nil)
	s.Apply(fish(1, event.KindIdle, "", "/a"), at(0))
	if got := s.Summary(at(1)); got.Level != LevelShellIdle {
		t.Fatalf("quero shell parado: %+v", got)
	}
	s.Apply(fish(2, event.KindExec, "nvim", "/b"), at(2))
	if got := s.Summary(at(3)); got.Level != LevelShellExec || got.Prog != "nvim" || got.Cwd != "/b" {
		t.Fatalf("quero comando rodando: %+v", got)
	}
	s.Apply(claude("s1", 0, event.KindBusy, "/c"), at(4))
	if got := s.Summary(at(5)); got.Level != LevelClaudeBusy || got.Cwd != "/c" || got.Count != 1 {
		t.Fatalf("quero Claude trabalhando: %+v", got)
	}
	s.Apply(claude("s1", 0, event.KindIdle, ""), at(6))
	if got := s.Summary(at(7)); got.Level != LevelShellExec {
		t.Fatalf("comando rodando fica acima do Claude esperando: %+v", got)
	}
	s.Apply(fish(2, event.KindIdle, "", "/b"), at(8))
	if got := s.Summary(at(9)); got.Level != LevelClaudeIdle || got.Cwd != "/c" {
		t.Fatalf("Claude esperando fica acima do shell parado: %+v", got)
	}
}

func TestFishRunningClaudeIsIgnored(t *testing.T) {
	s := newStore(nil)
	s.Apply(fish(1, event.KindExec, "claude", "/p"), at(0))
	if got := s.Summary(at(1)); got.Level != LevelNone {
		t.Fatalf("o fish rodando claude não deveria aparecer: %+v", got)
	}
	s.Apply(claude("s", 0, event.KindIdle, "/p"), at(1))
	if got := s.Summary(at(2)); got.Level != LevelClaudeIdle {
		t.Fatalf("quem representa o terminal é a fonte do Claude: %+v", got)
	}
}

func TestToolChangesKeepTimer(t *testing.T) {
	s := newStore(nil)
	s.Apply(claudeTool("s", ""), at(0))
	s.Apply(claudeTool("s", "Read"), at(5))
	s.Apply(claudeTool("s", "Edit"), at(10))
	got := s.Summary(at(11))
	if got.Tool != "Edit" || !got.Since.Equal(at(0)) {
		t.Fatalf("quero a ferramenta mais recente sem reiniciar o cronômetro: %+v", got)
	}
	s.Apply(claude("s", 0, event.KindIdle, ""), at(12))
	s.Apply(claudeTool("s", "Bash"), at(20))
	if got := s.Summary(at(21)); !got.Since.Equal(at(20)) {
		t.Fatalf("um prompt novo depois de esperar reinicia o cronômetro: %+v", got)
	}
}

func TestClaudeIdleTimeout(t *testing.T) {
	s := newStore(nil)
	s.Apply(claude("s", 0, event.KindIdle, "/p"), at(0))
	if got := s.Summary(at(11 * 60)); got.Level != LevelNone {
		t.Fatalf("Claude esperando demais deveria sumir: %+v", got)
	}
}

func TestAggregationOldestWins(t *testing.T) {
	s := newStore(nil)
	s.Apply(claude("novo", 0, event.KindBusy, "/novo"), at(10))
	s.Apply(claude("velho", 0, event.KindBusy, "/velho"), at(5))
	got := s.Summary(at(11))
	if got.Count != 2 || got.Cwd != "/velho" || !got.Since.Equal(at(5)) {
		t.Fatalf("quero 2 sessões com a pasta da mais antiga: %+v", got)
	}
}

func TestBusyKeepsTimerExecResets(t *testing.T) {
	s := newStore(nil)
	s.Apply(claude("s", 0, event.KindBusy, "/p"), at(0))
	s.Apply(claude("s", 0, event.KindBusy, "/p"), at(30))
	if got := s.Summary(at(31)); !got.Since.Equal(at(0)) {
		t.Fatalf("busy seguido não reinicia o cronômetro: %+v", got)
	}
	s = newStore(nil)
	s.Apply(fish(1, event.KindExec, "make", "/p"), at(0))
	s.Apply(fish(1, event.KindExec, "make", "/p"), at(30))
	if got := s.Summary(at(31)); !got.Since.Equal(at(30)) {
		t.Fatalf("comando novo reinicia o cronômetro: %+v", got)
	}
}

func TestEndRemoves(t *testing.T) {
	s := newStore(nil)
	s.Apply(fish(1, event.KindIdle, "", "/a"), at(0))
	s.Apply(fish(1, event.KindEnd, "", ""), at(1))
	if s.Len() != 0 {
		t.Fatal("end deveria remover a fonte")
	}
}

func TestPruneDeadPID(t *testing.T) {
	procs := fakeProcs{1: true, 2: true}
	s := newStore(procs)
	s.Apply(fish(1, event.KindExec, "nvim", "/a"), at(0))
	s.Apply(claude("s", 2, event.KindBusy, "/b"), at(0))
	procs[2] = false
	s.Prune(at(1))
	if got := s.Summary(at(1)); got.Level != LevelShellExec {
		t.Fatalf("Claude morto deveria sumir: %+v", got)
	}
	// Com PID, o TTL não se aplica: um nvim aberto por horas continua.
	s.Prune(at(5 * 3600))
	if s.Len() != 1 {
		t.Fatal("fonte com PID vivo não expira por TTL")
	}
}

func TestPruneTTLWithoutPID(t *testing.T) {
	s := newStore(nil)
	s.Apply(claude("s", 0, event.KindBusy, "/b"), at(0))
	s.Prune(at(9 * 60))
	if s.Len() != 1 {
		t.Fatal("ainda dentro do TTL")
	}
	s.Prune(at(11 * 60))
	if s.Len() != 0 {
		t.Fatal("deveria expirar pelo TTL")
	}
}

func TestIdleTimeout(t *testing.T) {
	s := newStore(fakeProcs{1: true})
	s.Apply(fish(1, event.KindIdle, "", "/a"), at(0))
	if got := s.Summary(at(9 * 60)); got.Level != LevelShellIdle {
		t.Fatalf("ainda dentro do limite: %+v", got)
	}
	if got := s.Summary(at(11 * 60)); got.Level != LevelNone {
		t.Fatalf("shell parado demais deveria limpar a presença: %+v", got)
	}
}
