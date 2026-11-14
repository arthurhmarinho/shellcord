package setup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// claudeEvents são os hooks que o shellcord usa (ADR-0014), na ordem em que
// entram no settings.json.
var claudeEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Notification", "Stop", "SessionEnd",
}

// toolEvents levam matcher, porque filtram por ferramenta.
var toolEvents = map[string]bool{"PreToolUse": true, "PostToolUse": true}

// object é um objeto JSON que preserva a ordem das chaves. O settings.json é do
// usuário: o shellcord mexe só nos próprios hooks e não reordena o resto.
type object []member

type member struct {
	key string
	val json.RawMessage
}

func decodeObject(b []byte) (object, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errors.New("não é um objeto JSON")
	}
	var o object
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		o = append(o, member{key: tok.(string), val: val})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err == nil {
		return nil, errors.New("conteúdo extra depois do objeto JSON")
	}
	return o, nil
}

func (o object) get(key string) (json.RawMessage, bool) {
	for _, m := range o {
		if m.key == key {
			return m.val, true
		}
	}
	return nil, false
}

func (o *object) set(key string, val json.RawMessage) {
	for i, m := range *o {
		if m.key == key {
			(*o)[i].val = val
			return
		}
	}
	*o = append(*o, member{key: key, val: val})
}

func (o *object) del(key string) {
	for i, m := range *o {
		if m.key == key {
			*o = append((*o)[:i], (*o)[i+1:]...)
			return
		}
	}
}

func (o object) marshal() (json.RawMessage, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, m := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := marshalNoEscape(m.key)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		if err := json.Compact(&buf, m.val); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// hookGroup é só o que interessa para reconhecer um grupo de hooks.
type hookGroup struct {
	Hooks []struct {
		Command string `json:"command"`
	} `json:"hooks"`
}

// isOurs diz se o grupo é do shellcord: todos os comandos são "… hook claude"
// de um binário shellcord.
func isOurs(raw json.RawMessage) bool {
	var g hookGroup
	if json.Unmarshal(raw, &g) != nil || len(g.Hooks) == 0 {
		return false
	}
	for _, h := range g.Hooks {
		if !isOurCommand(h.Command) {
			return false
		}
	}
	return true
}

func isOurCommand(cmd string) bool {
	return strings.HasSuffix(cmd, " hook claude") && strings.Contains(cmd, "shellcord")
}

// ourGroup monta o grupo de hooks do shellcord, com as chaves na ordem da
// documentação do Claude Code: matcher, hooks; type, command.
func ourGroup(event, exe string) (json.RawMessage, error) {
	cmd, err := marshalNoEscape(shellQuote(exe) + " hook claude")
	if err != nil {
		return nil, err
	}
	hook, err := object{{"type", json.RawMessage(`"command"`)}, {"command", cmd}}.marshal()
	if err != nil {
		return nil, err
	}
	var g object
	if toolEvents[event] {
		g = append(g, member{"matcher", json.RawMessage(`"*"`)})
	}
	g = append(g, member{"hooks", json.RawMessage("[" + string(hook) + "]")})
	return g.marshal()
}

// shellQuote só põe aspas quando o caminho precisa (o Claude Code roda o
// comando com sh -c).
func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t'\"\\$`;&|<>()*?[]#~!{}") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func planClaude(text, exe string, install bool) (string, error) {
	var root object
	if strings.TrimSpace(text) != "" {
		var err error
		if root, err = decodeObject([]byte(text)); err != nil {
			return "", fmt.Errorf("settings.json inválido: %w", err)
		}
	}
	var hooks object
	if raw, ok := root.get("hooks"); ok {
		var err error
		if hooks, err = decodeObject(raw); err != nil {
			return "", fmt.Errorf(`"hooks" do settings.json: %w`, err)
		}
	}
	for _, ev := range claudeEvents {
		var groups []json.RawMessage
		if raw, ok := hooks.get(ev); ok {
			if err := json.Unmarshal(raw, &groups); err != nil {
				return "", fmt.Errorf("hooks.%s não é uma lista: %w", ev, err)
			}
		}
		kept := groups[:0]
		for _, g := range groups {
			if !isOurs(g) {
				kept = append(kept, g)
			}
		}
		if install {
			g, err := ourGroup(ev, exe)
			if err != nil {
				return "", err
			}
			kept = append(kept, g)
		}
		if len(kept) == 0 {
			hooks.del(ev)
			continue
		}
		raw, err := json.Marshal(kept)
		if err != nil {
			return "", err
		}
		hooks.set(ev, raw)
	}
	if len(hooks) == 0 {
		root.del("hooks")
	} else {
		raw, err := hooks.marshal()
		if err != nil {
			return "", err
		}
		root.set("hooks", raw)
	}
	if len(root) == 0 && strings.TrimSpace(text) == "" {
		return text, nil
	}
	compact, err := root.marshal()
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact, "", "  "); err != nil {
		return "", err
	}
	out.WriteByte('\n')
	return out.String(), nil
}

// claudeMissing lista os hooks do shellcord que faltam no settings.json.
func claudeMissing(text string) ([]string, error) {
	root, err := decodeObject([]byte(text))
	if err != nil {
		return nil, fmt.Errorf("settings.json inválido: %w", err)
	}
	var hooks object
	if raw, ok := root.get("hooks"); ok {
		if hooks, err = decodeObject(raw); err != nil {
			return nil, err
		}
	}
	var missing []string
	for _, ev := range claudeEvents {
		found := false
		if raw, ok := hooks.get(ev); ok {
			var groups []json.RawMessage
			json.Unmarshal(raw, &groups)
			for _, g := range groups {
				found = found || isOurs(g)
			}
		}
		if !found {
			missing = append(missing, ev)
		}
	}
	return missing, nil
}
