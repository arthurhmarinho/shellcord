// Package state guarda as fontes de eventos e decide o que aparece no card:
// prioridade (ADR-0004), agregação (ADR-0011) e expiração (ADR-0010). Não faz
// E/S nem lê o relógio: quem chama passa o "agora", o que deixa os testes
// determinísticos.
package state

import (
	"time"

	"github.com/arthurhmarinho/shellcord/internal/event"
)

// Level é o nível de prioridade de uma fonte. Quanto maior, mais prioritário.
type Level int

const (
	LevelNone       Level = iota // não aparece
	LevelShellIdle               // shell parado no prompt
	LevelClaudeIdle              // Claude Code aberto, esperando você (ADR-0016)
	LevelShellExec               // comando rodando no terminal
	LevelClaudeBusy              // Claude Code trabalhando
)

// claudeProg é o programa que o fish reporta ao abrir o Claude Code. Essa fonte
// do fish é ignorada: a fonte do próprio Claude Code representa o terminal.
const claudeProg = "claude"

// Options configura o Store.
type Options struct {
	// TTL: fontes sem PID expiram depois desse tempo sem eventos.
	TTL time.Duration
	// IdleTimeout: um shell parado há mais que isso deixa de aparecer.
	IdleTimeout time.Duration
	// Alive diz se o processo da fonte continua vivo (normalmente proc.Alive).
	Alive func(pid int, start uint64) bool
}

// Summary é o que deve aparecer no card.
type Summary struct {
	Level Level
	Count int       // fontes no nível exibido
	Prog  string    // da fonte mais antiga do nível
	Tool  string    // da fonte mais antiga do nível (só Claude Code trabalhando)
	Cwd   string    // da fonte mais antiga do nível
	Since time.Time // desde quando a fonte mais antiga está nesse nível
}

type source struct {
	key   string
	pid   int
	start uint64
	level Level
	prog  string
	tool  string
	cwd   string
	since time.Time // quando entrou no nível atual
	seen  time.Time // último evento
}

// Store é o conjunto de fontes conhecidas. Não é seguro para uso concorrente.
type Store struct {
	opt     Options
	sources map[string]*source
	away    bool // o usuário está ausente (ADR-0019)
}

// New cria um Store vazio.
func New(opt Options) *Store {
	return &Store{opt: opt, sources: make(map[string]*source)}
}

// Len devolve o número de fontes conhecidas.
func (s *Store) Len() int { return len(s.sources) }

// Apply incorpora um evento já validado.
func (s *Store) Apply(e event.Event, now time.Time) {
	if e.Src == event.SourceSystem {
		s.away = e.Kind == event.KindAway
		return
	}
	key := e.Key()
	if e.Kind == event.KindEnd {
		delete(s.sources, key)
		return
	}
	src, ok := s.sources[key]
	if !ok {
		src = &source{key: key}
		s.sources[key] = src
	}
	src.pid, src.start, src.seen = e.PID, e.Start, now
	if e.Cwd != "" {
		src.cwd = e.Cwd
	}
	level := levelOf(e)
	// Um comando novo no fish reinicia o cronômetro. Nos outros casos, ele só
	// reinicia quando o nível muda: trocar de ferramenta no Claude Code continua
	// sendo o mesmo "trabalhando".
	if !ok || level != src.level || e.Kind == event.KindExec {
		src.since = now
	}
	src.level = level
	src.prog, src.tool = e.Prog, e.Tool
}

func levelOf(e event.Event) Level {
	switch {
	case e.Src == event.SourceFish && e.Kind == event.KindExec && e.Prog == claudeProg:
		return LevelNone
	case e.Src == event.SourceFish && e.Kind == event.KindExec:
		return LevelShellExec
	case e.Src == event.SourceFish && e.Kind == event.KindIdle:
		return LevelShellIdle
	case e.Src == event.SourceClaude && e.Kind == event.KindBusy:
		return LevelClaudeBusy
	case e.Src == event.SourceClaude && e.Kind == event.KindIdle:
		return LevelClaudeIdle
	}
	return LevelNone
}

// waiting diz se o nível é de espera, sujeito ao limite de inatividade.
func waiting(lv Level) bool { return lv == LevelShellIdle || lv == LevelClaudeIdle }

// Prune remove fontes mortas: com PID, quando o processo morreu; sem PID,
// quando passam do TTL sem eventos.
func (s *Store) Prune(now time.Time) {
	for k, src := range s.sources {
		if src.pid > 0 {
			if !s.opt.Alive(src.pid, src.start) {
				delete(s.sources, k)
			}
			continue
		}
		if now.Sub(src.seen) > s.opt.TTL {
			delete(s.sources, k)
		}
	}
}

// Summary calcula o que aparece: o nível mais alto, quantas fontes estão nele
// e os dados da mais antiga. Level == LevelNone significa presença limpa.
func (s *Store) Summary(now time.Time) Summary {
	var sum Summary
	var oldest *source
	for _, src := range s.sources {
		lv := src.level
		if waiting(lv) && s.opt.IdleTimeout > 0 && now.Sub(src.since) > s.opt.IdleTimeout {
			lv = LevelNone
		}
		// Ausente, só o Claude Code trabalhando continua: ele está mesmo
		// trabalhando, e o resto seria um terminal esquecido aberto.
		if s.away && lv != LevelClaudeBusy {
			lv = LevelNone
		}
		if lv == LevelNone || lv < sum.Level {
			continue
		}
		if lv > sum.Level {
			sum = Summary{Level: lv}
			oldest = nil
		}
		sum.Count++
		if oldest == nil || src.since.Before(oldest.since) ||
			(src.since.Equal(oldest.since) && src.key < oldest.key) {
			oldest = src
		}
	}
	if oldest != nil {
		sum.Prog, sum.Tool, sum.Cwd, sum.Since = oldest.prog, oldest.tool, oldest.cwd, oldest.since
	}
	return sum
}
