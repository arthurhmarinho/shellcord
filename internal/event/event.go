// Package event define o pacote que o cliente manda ao daemon pelo socket de
// eventos (ADR-0009): um objeto JSON por datagrama.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Version é a versão do formato do pacote. O daemon recusa outras versões.
const Version = 1

// MaxSize é o tamanho máximo de um pacote, em bytes.
const MaxSize = 4096

// Source identifica de onde vem o evento.
type Source string

const (
	SourceFish   Source = "fish"
	SourceClaude Source = "claude"
	SourceSystem Source = "system" // ausência do usuário, via swayidle (ADR-0019)
)

// Kind é o tipo do evento.
type Kind string

const (
	KindExec Kind = "exec" // fish: um comando começou
	KindIdle Kind = "idle" // fish: o prompt voltou; claude: parou de trabalhar
	KindBusy Kind = "busy" // claude: trabalhando
	KindEnd  Kind = "end"  // a fonte encerrou
	KindAway Kind = "away" // system: o usuário saiu (sem teclado nem mouse)
	KindBack Kind = "back" // system: o usuário voltou
)

// Event é um evento de uma fonte.
type Event struct {
	V     int    `json:"v"`
	Src   Source `json:"src"`
	ID    string `json:"id,omitempty"`    // session_id do Claude Code
	PID   int    `json:"pid,omitempty"`   // processo que representa a fonte (ADR-0010)
	Start uint64 `json:"start,omitempty"` // starttime desse processo, em ticks desde o boot
	Kind  Kind   `json:"kind"`
	Prog  string `json:"prog,omitempty"` // só o nome do programa, nunca argumentos (ADR-0005)
	Tool  string `json:"tool,omitempty"` // claude: tool_name em uso; o daemon traduz (ADR-0015)
	Cwd   string `json:"cwd,omitempty"`  // caminho completo; o daemon reduz (ADR-0012)
}

// Key identifica a fonte no estado do daemon.
func (e Event) Key() string {
	switch e.Src {
	case SourceClaude:
		return "claude:" + e.ID
	case SourceSystem:
		return "system"
	}
	return fmt.Sprintf("fish:%d", e.PID)
}

// Validate confere se o evento é coerente com a fonte.
func (e Event) Validate() error {
	if e.V != Version {
		return fmt.Errorf("versão %d não suportada", e.V)
	}
	switch e.Src {
	case SourceFish:
		if e.PID <= 0 {
			return errors.New("fish: pid é obrigatório")
		}
		switch e.Kind {
		case KindExec, KindIdle, KindEnd:
		default:
			return fmt.Errorf("fish: tipo %q inválido", e.Kind)
		}
		if e.Kind == KindExec && e.Prog == "" {
			return errors.New("fish: exec sem prog")
		}
	case SourceClaude:
		if e.ID == "" {
			return errors.New("claude: id (session_id) é obrigatório")
		}
		switch e.Kind {
		case KindBusy, KindIdle, KindEnd:
		default:
			return fmt.Errorf("claude: tipo %q inválido", e.Kind)
		}
		if e.Tool != "" && e.Kind != KindBusy {
			return errors.New("claude: tool só faz sentido com busy")
		}
	case SourceSystem:
		if e.Kind != KindAway && e.Kind != KindBack {
			return fmt.Errorf("system: tipo %q inválido", e.Kind)
		}
		if e.Prog != "" || e.Cwd != "" {
			return errors.New("system: evento não leva prog nem cwd")
		}
	default:
		return fmt.Errorf("fonte %q desconhecida", e.Src)
	}
	if e.Src != SourceClaude && e.Tool != "" {
		return errors.New("tool só existe para o claude")
	}
	if e.Prog != ProgName(e.Prog) {
		return errors.New("prog deve ser só o nome do programa")
	}
	return nil
}

// wrappers são comandos que só embrulham outro (ADR-0018). O valor lista as
// opções de uma letra que consomem a palavra seguinte (ex.: "sudo -u root").
var wrappers = map[string]string{
	"sudo": "ugpCDhrtU", "doas": "uC", "env": "uCS", "time": "fo", "nice": "n",
	"command": "", "builtin": "", "exec": "", "nohup": "",
}

// ProgName reduz uma linha de comando ao nome do programa: pula atribuições
// (VAR=valor) e wrappers com as opções deles, e fica com a primeira palavra
// que sobra, sem o caminho. Os argumentos são descartados (ADR-0005). Se só
// houver wrappers, devolve o último; se só houver atribuições, devolve "".
func ProgName(cmdline string) string {
	f := strings.Fields(cmdline)
	last := ""
	for i := 0; i < len(f); {
		if isAssign(f[i]) {
			i++
			continue
		}
		name := progBase(f[i])
		valued, ok := wrappers[name]
		if !ok {
			return name
		}
		last = name
		i++
		for i < len(f) {
			a := f[i]
			if a == "--" {
				i++
				break
			}
			if len(a) < 2 || a[0] != '-' {
				break
			}
			i++
			if len(a) == 2 && strings.IndexByte(valued, a[1]) >= 0 {
				i++ // a opção leva o valor na palavra seguinte
			}
		}
	}
	return last
}

func progBase(word string) string {
	return filepath.Base(strings.TrimRight(word, ";&|"))
}

// isAssign diz se a palavra é uma atribuição VAR=valor.
func isAssign(word string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 {
		return false
	}
	for i, c := range word[:eq] {
		if c != '_' && !(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Encode valida e serializa o evento.
func Encode(e Event) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxSize {
		return nil, fmt.Errorf("evento com %d bytes passa do limite de %d", len(b), MaxSize)
	}
	return b, nil
}

// Decode desserializa e valida um pacote. Campos desconhecidos são ignorados,
// para que um cliente mais novo não quebre um daemon mais velho.
func Decode(b []byte) (Event, error) {
	var e Event
	if err := json.Unmarshal(b, &e); err != nil {
		return Event{}, err
	}
	return e, e.Validate()
}
