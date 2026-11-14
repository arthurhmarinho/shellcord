package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/arthurhmarinho/shellcord/internal/event"
)

// maxHookInput limita quanto do stdin do hook é lido. O PostToolUse traz a
// resposta inteira da ferramenta, que pode ser grande; o resto é descartado.
const maxHookInput = 16 << 20

// claudeHookInput são os únicos campos lidos do JSON do hook. O prompt, o
// tool_input e a resposta do assistente também chegam no stdin, mas nem são
// decodificados (ADR-0014).
type claudeHookInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Event     string `json:"hook_event_name"`
	ToolName  string `json:"tool_name"`
}

// claudeKinds mapeia cada hook do Claude Code para um tipo de evento.
var claudeKinds = map[string]event.Kind{
	"SessionStart":     event.KindIdle,
	"UserPromptSubmit": event.KindBusy,
	"PreToolUse":       event.KindBusy,
	"PostToolUse":      event.KindBusy,
	"Notification":     event.KindIdle, // o Claude está esperando você
	"Stop":             event.KindIdle,
	"SessionEnd":       event.KindEnd,
}

// runHook trata "shellcord hook claude". Regras fixas (ADR-0014): nunca escreve
// no stdout, que alguns hooks injetam no contexto do Claude, e sempre sai com
// código 0, para um problema no shellcord não atrapalhar a sessão. Erros só
// aparecem no stderr com SHELLCORD_DEBUG=1.
func runHook(args []string) {
	defer func() {
		if r := recover(); r != nil {
			debugf("pânico no hook: %v", r)
		}
		os.Exit(0)
	}()
	if len(args) != 1 || args[0] != "claude" {
		debugf("uso: shellcord hook claude")
		return
	}
	e, ok, err := claudeEvent(os.Stdin)
	io.Copy(io.Discard, os.Stdin) // drena o resto, para o Claude Code não esbarrar num pipe fechado
	if err != nil {
		debugf("hook: %v", err)
		return
	}
	if !ok {
		return // hook que não interessa ao shellcord
	}
	fillPID(&e, 0)
	b, err := event.Encode(e)
	if err == nil {
		err = send(b)
	}
	if err != nil {
		debugf("hook: %v", err)
	}
}

// claudeEvent lê o JSON do hook e monta o evento. ok == false quando o hook
// não gera evento.
func claudeEvent(r io.Reader) (event.Event, bool, error) {
	var in claudeHookInput
	if err := json.NewDecoder(io.LimitReader(r, maxHookInput)).Decode(&in); err != nil {
		return event.Event{}, false, fmt.Errorf("JSON do hook: %w", err)
	}
	kind, ok := claudeKinds[in.Event]
	if !ok {
		return event.Event{}, false, nil
	}
	e := event.Event{V: event.Version, Src: event.SourceClaude, ID: in.SessionID, Kind: kind, Cwd: in.Cwd}
	if kind == event.KindBusy {
		e.Tool = in.ToolName
	}
	return e, true, nil
}

func debugf(format string, args ...any) {
	if os.Getenv("SHELLCORD_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "shellcord: "+format+"\n", args...)
	}
}
