package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exe = "/home/u/go/bin/shellcord"

// userSettings imita o settings.json real: chaves fora de ordem alfabética,
// texto não ASCII e os hooks do shellcord já instalados à mão (indent de 2).
const userSettings = `{
  "model": "opus",
  "theme": "auto",
  "autoMode": {
    "environment": [
      "**Organization**: None configured — nada"
    ]
  },
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "Notification": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/home/u/go/bin/shellcord hook claude"
          }
        ]
      }
    ]
  }
}
`

func TestClaudeInstallIsIdempotentOnUserFile(t *testing.T) {
	got, err := planClaude(userSettings, exe, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != userSettings {
		t.Fatalf("reinstalar não deveria mudar nada:\n%s", Diff(userSettings, got))
	}
}

func TestClaudeUninstallKeepsEverythingElse(t *testing.T) {
	got, err := planClaude(userSettings, exe, false)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "model": "opus",
  "theme": "auto",
  "autoMode": {
    "environment": [
      "**Organization**: None configured — nada"
    ]
  }
}
`
	if got != want {
		t.Fatalf("remover deveria tirar só os hooks:\n%s", Diff(want, got))
	}
	back, err := planClaude(got, exe, true)
	if err != nil || back != userSettings {
		t.Fatalf("instalar de novo deveria voltar ao original (%v):\n%s", err, Diff(userSettings, back))
	}
}

func TestClaudeKeepsOtherHooks(t *testing.T) {
	in := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"notify-send pronto"}]}]}}`
	got, err := planClaude(in, exe, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "notify-send pronto") || strings.Count(got, "hook claude") != 7 {
		t.Fatalf("o hook do usuário deve continuar e os 7 do shellcord entrar:\n%s", got)
	}
	if missing, _ := claudeMissing(got); len(missing) != 0 {
		t.Fatalf("faltando: %v", missing)
	}
	out, _ := planClaude(got, exe, false)
	if !strings.Contains(out, "notify-send pronto") || strings.Contains(out, "hook claude") {
		t.Fatalf("remover tira só os do shellcord:\n%s", out)
	}
}

func TestClaudeNewBinaryPathReplacesOld(t *testing.T) {
	got, err := planClaude(userSettings, "/opt/shellcord", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "/home/u/go/bin") || strings.Count(got, "/opt/shellcord hook claude") != 7 {
		t.Fatalf("o caminho antigo deveria ser trocado:\n%s", got)
	}
}

func TestClaudeEmptyAndInvalid(t *testing.T) {
	got, err := planClaude("", exe, true)
	if err != nil || strings.Count(got, "hook claude") != 7 {
		t.Fatalf("arquivo vazio: %v\n%s", err, got)
	}
	if _, err := planClaude(`{"model": `, exe, true); err == nil {
		t.Fatal("JSON quebrado deveria dar erro, nunca ser sobrescrito")
	}
	if _, err := planClaude(`{"hooks": {"Stop": {}}}`, exe, true); err == nil {
		t.Fatal("hooks.Stop que não é lista deveria dar erro")
	}
}

func TestShellQuote(t *testing.T) {
	if shellQuote(exe) != exe {
		t.Error("caminho simples não leva aspas")
	}
	if got := shellQuote("/tmp/meu dir/it's"); got != `'/tmp/meu dir/it'\''s'` {
		t.Errorf("shellQuote: %s", got)
	}
}

const fishConfig = `if status is-interactive
    starship init fish | source
end
`

func TestFishBlock(t *testing.T) {
	got, note, err := planBlock(fishConfig, fishBlock(exe), fishManual, true)
	if err != nil || note != "" {
		t.Fatal(err, note)
	}
	if !strings.HasPrefix(got, fishConfig+"\n"+beginMark) || !strings.Contains(got, "'"+exe+"' init fish | source") {
		t.Fatalf("bloco no fim do arquivo:\n%s", got)
	}
	again, _, _ := planBlock(got, fishBlock(exe), fishManual, true)
	if again != got {
		t.Fatalf("instalar duas vezes não muda nada:\n%s", Diff(got, again))
	}
	moved, _, _ := planBlock(got, fishBlock("/opt/shellcord"), fishManual, true)
	if strings.Count(moved, beginMark) != 1 || !strings.Contains(moved, "'/opt/shellcord'") {
		t.Fatalf("binário novo troca o bloco no lugar:\n%s", moved)
	}
	removed, _, _ := planBlock(got, "", fishManual, false)
	if removed != fishConfig {
		t.Fatalf("remover volta ao original:\n%s", Diff(fishConfig, removed))
	}
}

func TestFishManualIntegrationIsLeftAlone(t *testing.T) {
	manual := strings.Replace(fishConfig, "source\n", "source\n    test -x ~/go/bin/shellcord; and ~/go/bin/shellcord init fish | source\n", 1)
	got, note, err := planBlock(manual, fishBlock(exe), fishManual, true)
	if err != nil || got != manual || note == "" {
		t.Fatalf("integração à mão: não duplica e avisa (%v, %q)", err, note)
	}
	if ok, _ := Installed(Fish, writeFile(t, manual)); !ok {
		t.Fatal("Installed deveria reconhecer a integração à mão")
	}
}

func TestBrokenMarkers(t *testing.T) {
	if _, _, err := planBlock("x\n"+beginMark+"\ny\n", "b", fishManual, true); err == nil {
		t.Fatal("início sem fim deveria dar erro")
	}
	if _, _, err := planBlock("x\n"+endMark+"\n", "b", fishManual, true); err == nil {
		t.Fatal("fim sem início deveria dar erro")
	}
}

func TestSway(t *testing.T) {
	block, err := swayBlock(exe)
	if err != nil {
		t.Fatal(err)
	}
	conf := "exec noctalia\n"
	got, _, _ := planBlock(conf, block, swayManual, true)
	if ok, why := Installed(Sway, writeFile(t, got)); !ok {
		t.Fatalf("depois de instalar, Installed deveria ver o sway completo: %s", why)
	}
	if _, err := swayBlock("/home/meu usuario/shellcord"); err == nil {
		t.Fatal("caminho com espaço deveria ser recusado no sway")
	}
	if _, err := Plan(Sway, filepath.Join(t.TempDir(), "nao-existe"), exe, true); err == nil {
		t.Fatal("sem config do sway não cria uma do zero")
	}
}

func writeFile(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestApplyFollowsSymlinkAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "dotfiles-config")
	link := filepath.Join(dir, "config")
	if err := os.WriteFile(real, []byte("velho\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	backup, err := Apply(Change{Path: link, Old: "velho\n", New: "novo\n", Exists: true})
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("o symlink deveria continuar symlink")
	}
	if b, _ := os.ReadFile(real); string(b) != "novo\n" {
		t.Fatalf("o alvo do symlink deveria ter o texto novo: %q", b)
	}
	if fi, _ := os.Stat(real); fi.Mode().Perm() != 0o600 {
		t.Fatalf("a permissão deveria ser mantida: %v", fi.Mode().Perm())
	}
	if b, _ := os.ReadFile(backup); string(b) != "velho\n" {
		t.Fatalf("backup: %q", b)
	}
}

func TestDiff(t *testing.T) {
	d := Diff("a\nb\nc\n", "a\nB\nc\nd\n")
	want := "  a\n- b\n+ B\n  c\n+ d\n"
	if d != want {
		t.Fatalf("diff:\n%s\nquero:\n%s", d, want)
	}
}
