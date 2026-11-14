package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arthurhmarinho/shellcord/internal/setup"
)

// runInstall trata "shellcord install" e "shellcord uninstall" (ADR-0022):
// planeja, mostra o diff de cada arquivo, pede confirmação e só então grava,
// com backup em <arquivo>.shellcord.bak.
func runInstall(name string, args []string) error {
	install := name == "install"
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	yes := fs.Bool("yes", false, "aplica sem perguntar")
	dry := fs.Bool("dry-run", false, "só mostra o diff, não grava nada")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "uso: shellcord %s [-yes] [-dry-run] [claude] [fish] [sway]\n\nSem alvos, vale para os três.\n", name)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	targets, err := parseTargets(fs.Args())
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}

	var changes []setup.Change
	for _, t := range targets {
		c, err := setup.Plan(t, setup.Path(t), exe, install)
		if err != nil {
			return fmt.Errorf("%s: %w", t, err)
		}
		if c.Old == c.New {
			msg := "nada a mudar"
			if c.Note != "" {
				msg = c.Note
			}
			fmt.Printf("%s (%s): %s\n", t, c.Path, msg)
			continue
		}
		fmt.Printf("\n%s (%s):\n%s", t, c.Path, setup.Diff(c.Old, c.New))
		if c.Note != "" {
			fmt.Printf("  obs.: %s\n", c.Note)
		}
		changes = append(changes, c)
	}
	if len(changes) == 0 || *dry {
		return nil
	}
	if !*yes && !confirm("\nAplicar estas mudanças? [s/N] ") {
		fmt.Println("Nada foi alterado.")
		return nil
	}
	for _, c := range changes {
		backup, err := setup.Apply(c)
		if err != nil {
			return fmt.Errorf("%s: %w", c.Path, err)
		}
		if backup != "" {
			fmt.Printf("✓ %s (backup em %s)\n", c.Path, backup)
		} else {
			fmt.Printf("✓ %s (criado)\n", c.Path)
		}
	}
	if install {
		printNextSteps(changes, exe)
	}
	return nil
}

func parseTargets(args []string) ([]setup.Target, error) {
	if len(args) == 0 {
		return setup.Targets, nil
	}
	var out []setup.Target
	for _, a := range args {
		t := setup.Target(a)
		switch t {
		case setup.Claude, setup.Fish, setup.Sway:
			out = append(out, t)
		default:
			return nil, fmt.Errorf("alvo desconhecido %q (use claude, fish ou sway)", a)
		}
	}
	return out, nil
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "s", "sim", "y", "yes":
		return true
	}
	return false
}

func printNextSteps(changes []setup.Change, exe string) {
	fmt.Println("\nPróximos passos:")
	for _, c := range changes {
		switch c.Target {
		case setup.Claude:
			fmt.Println("  claude: vale nas sessões novas do Claude Code.")
		case setup.Fish:
			fmt.Println("  fish:   vale nos terminais novos.")
		case setup.Sway:
			fmt.Println("  sway:   vale na próxima sessão do sway. Para valer agora, sem reiniciar:")
			fmt.Printf("          setsid -f swayidle -w timeout 600 '%[1]s away' resume '%[1]s back'\n", exe)
		}
	}
	fmt.Println("  Para conferir tudo: shellcord doctor")
}
