// Package setup instala e remove as integrações do shellcord nos arquivos do
// usuário: hooks do Claude Code, fish e sway (ADR-0022). Tudo é planejado
// primeiro (texto antigo e novo) para quem chama mostrar o diff e confirmar.
package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Target é um arquivo que o shellcord sabe integrar.
type Target string

const (
	Claude Target = "claude"
	Fish   Target = "fish"
	Sway   Target = "sway"
)

// Targets são todos os alvos, na ordem em que são instalados.
var Targets = []Target{Claude, Fish, Sway}

// Path devolve o arquivo de cada alvo.
func Path(t Target) string {
	home, _ := os.UserHomeDir()
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	switch t {
	case Claude:
		dir := os.Getenv("CLAUDE_CONFIG_DIR")
		if dir == "" {
			dir = filepath.Join(home, ".claude")
		}
		return filepath.Join(dir, "settings.json")
	case Fish:
		return filepath.Join(cfg, "fish", "config.fish")
	case Sway:
		return filepath.Join(cfg, "sway", "config")
	}
	return ""
}

// Change é a mudança planejada num arquivo. Old == New significa que não há
// nada a fazer.
type Change struct {
	Target Target
	Path   string
	Old    string
	New    string
	Exists bool   // o arquivo existe hoje
	Note   string // observação para mostrar ao usuário
}

// Plan calcula a mudança para instalar (install == true) ou remover a
// integração do alvo, usando exe como caminho do binário.
func Plan(t Target, path, exe string, install bool) (Change, error) {
	c := Change{Target: t, Path: path}
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		c.Exists = true
		c.Old = string(b)
	case errors.Is(err, os.ErrNotExist):
		if t == Sway && install {
			return c, fmt.Errorf("%s não existe: o sway usa outra config?", path)
		}
	default:
		return c, err
	}
	switch t {
	case Claude:
		c.New, err = planClaude(c.Old, exe, install)
	case Fish:
		c.New, c.Note, err = planBlock(c.Old, fishBlock(exe), fishManual, install)
	case Sway:
		var block string
		if block, err = swayBlock(exe); err == nil {
			c.New, c.Note, err = planBlock(c.Old, block, swayManual, install)
		}
	default:
		err = fmt.Errorf("alvo desconhecido %q", t)
	}
	return c, err
}

const (
	beginMark = "# >>> shellcord >>>"
	endMark   = "# <<< shellcord <<<"
)

var (
	fishManual = regexp.MustCompile(`shellcord'? init fish`)
	swayManual = regexp.MustCompile(`shellcord'? daemon`)
)

func fishBlock(exe string) string {
	q := fishQuote(exe)
	return beginMark + `
# Integração do shellcord com o fish. Gerada por "shellcord install fish".
if status is-interactive; and test -x ` + q + `
    ` + q + ` init fish | source
end
` + endMark + "\n"
}

func swayBlock(exe string) (string, error) {
	if strings.ContainsAny(exe, " \t'\"\\") {
		return "", fmt.Errorf("caminho do binário com espaço ou aspas não é suportado no sway: %s", exe)
	}
	return beginMark + `
# shellcord: Discord Rich Presence do terminal e do Claude Code. Gerado por
# "shellcord install sway". O daemon segura a conexao com o Discord, e o swayidle
# esconde o card depois de 10 min sem teclado nem mouse.
exec ` + exe + ` daemon
exec swayidle -w timeout 600 '` + exe + ` away' resume '` + exe + ` back'
` + endMark + "\n", nil
}

// planBlock põe ou tira o bloco entre os marcadores. Uma integração feita à mão,
// fora dos marcadores, não é tocada: instalar de novo duplicaria os eventos.
func planBlock(old, block string, manual *regexp.Regexp, install bool) (string, string, error) {
	before, _, after, found, err := splitBlock(old)
	if err != nil {
		return "", "", err
	}
	if manual.MatchString(before + after) {
		if install {
			return old, "já integrado à mão, fora dos marcadores: nada a fazer", nil
		}
		if !found {
			return old, "a integração feita à mão, fora dos marcadores, não é removida automaticamente", nil
		}
	}
	if !install {
		if !found {
			return old, "", nil
		}
		before, after = strings.TrimRight(before, "\n"), strings.TrimLeft(after, "\n")
		switch {
		case before == "":
			return after, "", nil
		case after == "":
			return before + "\n", "", nil
		}
		return before + "\n\n" + after, "", nil
	}
	if found {
		return before + block + after, "", nil
	}
	if old != "" && !strings.HasSuffix(old, "\n") {
		old += "\n"
	}
	if old != "" {
		old += "\n"
	}
	return old + block, "", nil
}

// splitBlock separa o texto em antes, bloco e depois dos marcadores.
func splitBlock(text string) (before, inside, after string, found bool, err error) {
	i := strings.Index(text, beginMark)
	if i < 0 {
		if strings.Contains(text, endMark) {
			return "", "", "", false, errors.New("marcador de fim do shellcord sem o de início")
		}
		return text, "", "", false, nil
	}
	j := strings.Index(text[i:], endMark)
	if j < 0 {
		return "", "", "", false, errors.New("marcador de início do shellcord sem o de fim")
	}
	end := i + j + len(endMark)
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return text[:i], text[i:end], text[end:], true, nil
}

// fishQuote põe a palavra entre aspas simples, no formato do fish.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

// Apply grava a mudança: copia o original para <arquivo>.shellcord.bak e
// escreve o novo de forma atômica, seguindo symlinks (dotfiles gerenciados
// continuam como symlink). Devolve o caminho do backup, se houve.
func Apply(c Change) (string, error) {
	if c.Old == c.New {
		return "", nil
	}
	path := c.Path
	if p, err := filepath.EvalSymlinks(path); err == nil {
		path = p
	}
	mode := os.FileMode(0o644)
	backup := ""
	if c.Exists {
		st, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		mode = st.Mode().Perm()
		backup = path + ".shellcord.bak"
		if err := os.WriteFile(backup, []byte(c.Old), mode); err != nil {
			return "", err
		}
	} else if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".shellcord-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(c.New); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return backup, os.Rename(tmp.Name(), path)
}

// Installed diz se o alvo tem a integração (em bloco ou feita à mão) e, se
// não tiver por inteiro, o que falta.
func Installed(t Target, path string) (bool, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, "arquivo não encontrado: " + path
	}
	text := string(b)
	switch t {
	case Claude:
		missing, err := claudeMissing(text)
		if err != nil {
			return false, err.Error()
		}
		if len(missing) > 0 {
			return false, "hooks faltando: " + strings.Join(missing, ", ")
		}
		return true, ""
	case Fish:
		if fishManual.MatchString(text) {
			return true, ""
		}
		return false, `sem "shellcord init fish"`
	case Sway:
		hasDaemon := swayManual.MatchString(text)
		hasIdle := regexp.MustCompile(`swayidle.*shellcord'? away`).MatchString(text)
		switch {
		case hasDaemon && hasIdle:
			return true, ""
		case !hasDaemon && !hasIdle:
			return false, "sem o daemon e sem o swayidle"
		case !hasDaemon:
			return false, `sem "exec shellcord daemon"`
		default:
			return false, "sem o swayidle chamando away/back"
		}
	}
	return false, "alvo desconhecido"
}
