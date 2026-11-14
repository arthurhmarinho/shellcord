package main

import (
	"runtime/debug"
	"strings"
)

// version lê a versão que o próprio Go embute no binário (ADR-0024): a tag do
// módulo num "go install …@vX", ou a revisão do git num build local.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "desconhecida"
	}
	v := bi.Main.Version
	var rev, modified string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if (v == "" || v == "(devel)") && rev != "" {
		v = "dev " + rev[:min(12, len(rev))]
		if modified == "true" {
			v += " (com mudanças não commitadas)"
		}
	}
	return strings.TrimSpace(v)
}
