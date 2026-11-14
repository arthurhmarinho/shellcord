// Package config lê ~/.config/shellcord/config.toml (ADR-0005, ADR-0020).
// Só o daemon lê a config; o cliente não depende dela.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// DefaultTOML é a config padrão, exatamente como "shellcord config default" a
// imprime. Um teste garante que ela equivale a Default().
const DefaultTOML = `# Config do shellcord: ~/.config/shellcord/config.toml
# O daemon relê este arquivo em até 15 s depois de salvo. Para validar antes:
#   shellcord config check

# Programas que podem aparecer pelo nome ("Rodando nvim"). Os outros aparecem só
# como "No terminal". Esta lista SUBSTITUI a padrão: copie o que quiser manter.
allow = [
  "nvim", "vim", "hx", "nano",
  "git", "lazygit",
  "go", "cargo", "make",
  "npm", "pnpm", "node", "python", "python3",
  "docker", "htop", "btop", "man",
]

# Pastas ignoradas. Um terminal ou sessão do Claude Code dentro de uma delas (ou
# de uma subpasta) não aparece no card de jeito nenhum. Aceita "~".
ignore_dirs = []

# Minutos com o shell parado no prompt, ou o Claude Code esperando você, até o
# card sumir.
idle_minutes = 10
`

// Config é a config já validada e normalizada.
type Config struct {
	Allow       []string
	IgnoreDirs  []string // caminhos absolutos e limpos, com "~" expandido
	IdleTimeout time.Duration
}

type file struct {
	Allow       []string `toml:"allow"`
	IgnoreDirs  []string `toml:"ignore_dirs"`
	IdleMinutes int      `toml:"idle_minutes"`
}

// Default devolve a config usada quando o arquivo não existe.
func Default() Config {
	c, err := parse(DefaultTOML, "")
	if err != nil {
		panic("config padrão inválida: " + err.Error())
	}
	return c
}

// Path é o caminho da config: $XDG_CONFIG_HOME/shellcord/config.toml, ou
// ~/.config/shellcord/config.toml.
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "shellcord", "config.toml")
}

// Load lê e valida a config. Arquivo inexistente devolve Default() sem erro.
// Qualquer outro problema é erro: quem chama decide falhar fechado (ADR-0020).
func Load(path, home string) (Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, err
	}
	c, err := parse(string(b), home)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

func parse(text, home string) (Config, error) {
	var f file
	md, err := toml.Decode(text, &f)
	if err != nil {
		return Config{}, err
	}
	// Chave desconhecida é erro: um "ignore_dir" com erro de digitação não pode
	// passar em silêncio e deixar a pasta aparecer.
	if und := md.Undecoded(); len(und) > 0 {
		keys := make([]string, len(und))
		for i, k := range und {
			keys[i] = k.String()
		}
		return Config{}, fmt.Errorf("chave desconhecida: %s", strings.Join(keys, ", "))
	}

	var c Config
	if md.IsDefined("allow") {
		c.Allow = f.Allow
	} else {
		c.Allow = Default().Allow
	}
	for _, p := range c.Allow {
		if p == "" || strings.ContainsAny(p, "/ \t") {
			return Config{}, fmt.Errorf("allow: %q não é um nome de programa", p)
		}
	}

	for _, d := range f.IgnoreDirs {
		abs, err := expand(d, home)
		if err != nil {
			return Config{}, fmt.Errorf("ignore_dirs: %w", err)
		}
		c.IgnoreDirs = append(c.IgnoreDirs, abs)
	}

	c.IdleTimeout = 10 * time.Minute
	if md.IsDefined("idle_minutes") {
		if f.IdleMinutes < 1 {
			return Config{}, fmt.Errorf("idle_minutes deve ser pelo menos 1, veio %d", f.IdleMinutes)
		}
		c.IdleTimeout = time.Duration(f.IdleMinutes) * time.Minute
	}
	return c, nil
}

func expand(dir, home string) (string, error) {
	switch {
	case dir == "~":
		dir = home
	case strings.HasPrefix(dir, "~/"):
		dir = filepath.Join(home, dir[2:])
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("%q não é um caminho absoluto", dir)
	}
	return filepath.Clean(dir), nil
}

// AllowSet devolve a allowlist como conjunto.
func (c Config) AllowSet() map[string]bool {
	m := make(map[string]bool, len(c.Allow))
	for _, p := range c.Allow {
		m[p] = true
	}
	return m
}

// Ignored diz se cwd está numa pasta ignorada ou numa subpasta dela.
func (c Config) Ignored(cwd string) bool {
	if cwd == "" {
		return false
	}
	cwd = filepath.Clean(cwd)
	return slices.ContainsFunc(c.IgnoreDirs, func(d string) bool {
		return d == "/" || cwd == d || strings.HasPrefix(cwd, d+"/")
	})
}
