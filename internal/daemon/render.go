package daemon

import (
	"fmt"
	"path/filepath"

	"github.com/arthurhmarinho/shellcord/internal/discord"
	"github.com/arthurhmarinho/shellcord/internal/state"
)

// defaultAllow é a allowlist provisória de programas (ADR-0005). Fora dela, o
// card mostra só "No terminal". A fase 5 move isto para a config TOML.
var defaultAllow = map[string]bool{
	"nvim": true, "vim": true, "hx": true, "nano": true,
	"git": true, "lazygit": true,
	"go": true, "cargo": true, "make": true,
	"npm": true, "pnpm": true, "node": true, "python": true, "python3": true,
	"docker": true, "htop": true, "btop": true, "man": true,
}

// claudeVerbs traduz o tool_name do Claude Code no que aparece no card
// (ADR-0015). Ferramentas fora da tabela, inclusive as de MCP, viram
// "trabalhando": o nome delas nunca chega ao Discord.
var claudeVerbs = map[string]string{
	"Edit": "editando arquivos", "MultiEdit": "editando arquivos",
	"Write": "editando arquivos", "NotebookEdit": "editando arquivos",
	"Read": "lendo código", "Grep": "lendo código", "Glob": "lendo código",
	"Bash":     "rodando comandos",
	"WebFetch": "pesquisando na web", "WebSearch": "pesquisando na web",
}

func claudeVerb(tool string) string {
	if v, ok := claudeVerbs[tool]; ok {
		return v
	}
	return "trabalhando"
}

// maxField é o limite do Discord para details e state.
const maxField = 128

// render monta o card a partir do resumo. Devolve nil para limpar a presença.
// É aqui, e só aqui, que o caminho completo vira o nome da pasta (ADR-0012).
func render(s state.Summary, home string, allow map[string]bool) *discord.Activity {
	var details, unit string
	switch s.Level {
	case state.LevelClaudeBusy:
		details, unit = "Claude Code "+claudeVerb(s.Tool), "sessões"
	case state.LevelClaudeIdle:
		details, unit = "Claude Code esperando você", "sessões"
	case state.LevelShellExec:
		// Um programa fora da allowlist fica igual ao shell parado, para não
		// revelar nem que existe algo diferente rodando.
		details, unit = "No terminal", "terminais"
		if allow[s.Prog] {
			details = "Rodando " + s.Prog
		}
	case state.LevelShellIdle:
		details, unit = "No terminal", "terminais"
	default:
		return nil
	}
	if s.Count > 1 {
		details = fmt.Sprintf("%s · %d %s", details, s.Count, unit)
	}
	a := &discord.Activity{
		Details:    truncate(details),
		Timestamps: &discord.Timestamps{Start: s.Since.Unix()},
	}
	if dir := dirName(s.Cwd, home); dir != "" {
		a.State = truncate("em " + dir)
	}
	return a
}

// dirName reduz um caminho ao último componente; a home vira "~".
func dirName(cwd, home string) string {
	if cwd == "" {
		return ""
	}
	cwd = filepath.Clean(cwd)
	if home != "" && cwd == filepath.Clean(home) {
		return "~"
	}
	return filepath.Base(cwd)
}

func truncate(s string) string {
	r := []rune(s)
	if len(r) <= maxField {
		return s
	}
	return string(r[:maxField-1]) + "…"
}
