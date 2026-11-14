package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/arthurhmarinho/shellcord/internal/config"
)

const configUsage = `uso: shellcord config <check|default|path>

  check     valida a config e mostra o que vale
  default   imprime a config padrão (ponto de partida para editar)
  path      mostra onde a config fica`

func runConfig(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%s", configUsage)
	}
	switch args[0] {
	case "path":
		fmt.Println(config.Path())
	case "default":
		fmt.Print(config.DefaultTOML)
	case "check":
		home, _ := os.UserHomeDir()
		path := config.Path()
		c, err := config.Load(path, home)
		if err != nil {
			return fmt.Errorf("config inválida (o daemon não publica nada até corrigir): %w", err)
		}
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("%s não existe: vale a config padrão.\n", path)
		} else {
			fmt.Printf("%s: ok\n", path)
		}
		fmt.Printf("  allow:        %s\n", strings.Join(c.Allow, ", "))
		fmt.Printf("  ignore_dirs:  %s\n", strings.Join(c.IgnoreDirs, ", "))
		fmt.Printf("  idle_minutes: %d\n", int(c.IdleTimeout.Minutes()))
	default:
		return fmt.Errorf("%s", configUsage)
	}
	return nil
}
