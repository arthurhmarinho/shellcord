package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fishScript é a integração com o fish (ADR-0017). Os envios são síncronos,
// para os eventos chegarem em ordem. A linha de comando vai por variável de
// ambiente, não por argumento: /proc/<pid>/cmdline é legível por qualquer
// usuário da máquina, e /proc/<pid>/environ só pelo dono.
const fishScript = `# shellcord: integração com o fish. Gerado por "shellcord init fish".
function __shellcord_preexec --on-event fish_preexec
    SHELLCORD_CMDLINE=$argv[1] command %[1]s send -src fish -kind exec -pid $fish_pid -cwd "$PWD" 2>/dev/null
end
function __shellcord_postexec --on-event fish_postexec
    command %[1]s send -src fish -kind idle -pid $fish_pid -cwd "$PWD" 2>/dev/null
end
function __shellcord_exit --on-event fish_exit
    command %[1]s send -src fish -kind end -pid $fish_pid 2>/dev/null
end
command %[1]s send -src fish -kind idle -pid $fish_pid -cwd "$PWD" 2>/dev/null
`

func runInit(args []string) error {
	if len(args) != 1 || args[0] != "fish" {
		return fmt.Errorf("uso: shellcord init fish")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	fmt.Printf(fishScript, fishQuote(exe))
	return nil
}

// fishQuote põe a palavra entre aspas simples, no formato do fish.
func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}
