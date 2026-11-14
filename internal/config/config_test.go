package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMissingFileIsDefault(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nao-existe.toml"), "/h")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Allow, Default().Allow) || c.IdleTimeout != 10*time.Minute || len(c.IgnoreDirs) != 0 {
		t.Fatalf("sem arquivo deveria valer o padrão: %+v", c)
	}
}

func TestDefaultTOMLHasDefaults(t *testing.T) {
	c := Default()
	for _, p := range []string{"nvim", "git", "htop"} {
		if !c.AllowSet()[p] {
			t.Errorf("%s deveria estar na allowlist padrão", p)
		}
	}
}

func TestLoad(t *testing.T) {
	p := write(t, `
allow = ["nvim", "cargo"]
ignore_dirs = ["~/clientes", "/srv/segredo/", "~"]
idle_minutes = 3
`)
	c, err := Load(p, "/home/x")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Allow, []string{"nvim", "cargo"}) {
		t.Errorf("allow substitui a padrão: %v", c.Allow)
	}
	if !slices.Equal(c.IgnoreDirs, []string{"/home/x/clientes", "/srv/segredo", "/home/x"}) {
		t.Errorf("ignore_dirs: %v", c.IgnoreDirs)
	}
	if c.IdleTimeout != 3*time.Minute {
		t.Errorf("idle: %v", c.IdleTimeout)
	}
}

func TestAllowEmptyListIsAllowed(t *testing.T) {
	c, err := Load(write(t, `allow = []`), "/h")
	if err != nil || len(c.Allow) != 0 {
		t.Fatalf("allow vazio deveria esconder todos os programas: %+v %v", c, err)
	}
}

func TestInvalid(t *testing.T) {
	cases := map[string]string{
		`ignore_dir = ["/x"]`:        "chave desconhecida",
		`allow = "nvim"`:             "",
		`allow = ["git push"]`:       "não é um nome de programa",
		`allow = ["/usr/bin/nvim"]`:  "não é um nome de programa",
		`ignore_dirs = ["clientes"]`: "não é um caminho absoluto",
		`idle_minutes = 0`:           "pelo menos 1",
		`isso não é toml`:            "",
	}
	for text, want := range cases {
		_, err := Load(write(t, text), "/h")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: quero erro com %q, veio %v", text, want, err)
		}
	}
}

func TestIgnored(t *testing.T) {
	c := Config{IgnoreDirs: []string{"/h/clientes", "/srv"}}
	cases := map[string]bool{
		"/h/clientes":          true,
		"/h/clientes/acme/app": true,
		"/h/clientes/":         true,
		"/h/clientes-publicos": false,
		"/h":                   false,
		"/srv/x":               true,
		"":                     false,
	}
	for cwd, want := range cases {
		if got := c.Ignored(cwd); got != want {
			t.Errorf("Ignored(%q) = %v, quero %v", cwd, got, want)
		}
	}
	if !(Config{IgnoreDirs: []string{"/"}}).Ignored("/qualquer") {
		t.Error("ignorar / ignora tudo")
	}
}
