package main

import (
	"strings"
	"testing"

	"github.com/arthurhmarinho/shellcord/internal/event"
)

// Payloads reduzidos dos capturados na verificação da fase 3 (ADR-0010).
func TestClaudeEvent(t *testing.T) {
	cases := []struct {
		in   string
		kind event.Kind
		tool string
	}{
		{`{"session_id":"s","cwd":"/p","hook_event_name":"SessionStart","source":"startup"}`, event.KindIdle, ""},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"UserPromptSubmit","prompt":"segredo"}`, event.KindBusy, ""},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"echo segredo"}}`, event.KindBusy, "Bash"},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"PostToolUse","tool_name":"Edit","tool_response":{"x":1}}`, event.KindBusy, "Edit"},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"Notification","message":"permissão"}`, event.KindIdle, ""},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"Stop","last_assistant_message":"segredo"}`, event.KindIdle, ""},
		{`{"session_id":"s","cwd":"/p","hook_event_name":"SessionEnd","reason":"other"}`, event.KindEnd, ""},
	}
	for _, c := range cases {
		e, ok, err := claudeEvent(strings.NewReader(c.in))
		if err != nil || !ok {
			t.Fatalf("%s: ok=%v err=%v", c.in, ok, err)
		}
		if e.Kind != c.kind || e.Tool != c.tool || e.ID != "s" || e.Cwd != "/p" || e.Src != event.SourceClaude {
			t.Errorf("%s → %+v", c.in, e)
		}
		if err := e.Validate(); err != nil {
			t.Errorf("%s gerou evento inválido: %v", c.in, err)
		}
	}
}

func TestClaudeEventIgnoresUnknownHook(t *testing.T) {
	_, ok, err := claudeEvent(strings.NewReader(`{"session_id":"s","hook_event_name":"PreCompact"}`))
	if err != nil || ok {
		t.Fatalf("hook desconhecido deveria ser ignorado: ok=%v err=%v", ok, err)
	}
}

func TestClaudeEventBadJSON(t *testing.T) {
	if _, _, err := claudeEvent(strings.NewReader(`não é json`)); err == nil {
		t.Fatal("JSON inválido deveria dar erro")
	}
}

func TestFishQuote(t *testing.T) {
	cases := map[string]string{
		"/home/a/go/bin/shellcord": `'/home/a/go/bin/shellcord'`,
		`/tmp/it's here`:           `'/tmp/it\'s here'`,
		`/tmp/a\b`:                 `'/tmp/a\\b'`,
	}
	for in, want := range cases {
		if got := fishQuote(in); got != want {
			t.Errorf("fishQuote(%q) = %s, quero %s", in, got, want)
		}
	}
}
