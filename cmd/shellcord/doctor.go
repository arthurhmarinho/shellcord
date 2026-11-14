package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/arthurhmarinho/shellcord/internal/config"
	"github.com/arthurhmarinho/shellcord/internal/daemon"
	"github.com/arthurhmarinho/shellcord/internal/discord"
	"github.com/arthurhmarinho/shellcord/internal/setup"
)

// runDoctor confere só fatos externos ao daemon (ADR-0023): binário, Discord,
// daemon vivo, config e integrações, e mostra o fim do log.
func runDoctor(args []string) error {
	if len(args) != 0 {
		return errors.New("uso: shellcord doctor")
	}
	problems := 0
	check := func(ok bool, label, detail string) {
		mark := "✓"
		if !ok {
			mark = "✗"
			problems++
		}
		if detail != "" {
			label += ": " + detail
		}
		fmt.Printf("%s %s\n", mark, label)
	}
	info := func(label string) { fmt.Printf("· %s\n", label) }

	exe, _ := os.Executable()
	info(fmt.Sprintf("binário %s (%s)", exe, version()))
	if p, err := exec.LookPath("shellcord"); err != nil {
		info("shellcord não está no PATH; as integrações usam o caminho absoluto, então tudo bem")
	} else if p != exe {
		info("o shellcord do PATH é outro: " + p)
	}

	id := os.Getenv("SHELLCORD_CLIENT_ID")
	if id == "" {
		id = defaultClientID
	}
	if c, err := discord.Dial(id, discord.DefaultDirs()); err != nil {
		check(false, "Discord", err.Error())
	} else {
		c.Close()
		check(true, "Discord aberto e aceitando o handshake", "")
	}

	dir := daemon.RuntimeDir()
	running, err := daemonRunning(dir)
	switch {
	case err != nil:
		check(false, "daemon", err.Error())
	case running:
		check(true, "daemon rodando", "")
	default:
		check(false, "daemon", `parado (o próximo evento sobe ele, ou rode "shellcord daemon")`)
	}

	home, _ := os.UserHomeDir()
	if _, err := config.Load(config.Path(), home); err != nil {
		check(false, "config", err.Error()+" (o daemon não publica nada até corrigir)")
	} else if _, err := os.Stat(config.Path()); err != nil {
		check(true, "config", "sem arquivo, valem os padrões")
	} else {
		check(true, "config válida", config.Path())
	}

	for _, t := range setup.Targets {
		ok, why := setup.Installed(t, setup.Path(t))
		if !ok {
			why += fmt.Sprintf(` (rode "shellcord install %s")`, t)
		}
		check(ok, "integração "+string(t), why)
	}
	check(processRunning("swayidle"), "swayidle rodando", "")

	if lines := tail(filepath.Join(dir, "daemon.log"), 5); len(lines) > 0 {
		fmt.Println("\nÚltimas linhas do daemon.log:")
		for _, l := range lines {
			fmt.Println("  " + l)
		}
	}
	if problems > 0 {
		return fmt.Errorf("%d problema(s) encontrado(s)", problems)
	}
	fmt.Println("\nTudo certo.")
	return nil
}

// daemonRunning testa o lock do daemon: se conseguir pegar, não há daemon.
func daemonRunning(dir string) (bool, error) {
	f, err := os.Open(filepath.Join(dir, "daemon.lock"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false, nil
}

// processRunning procura um processo do usuário com esse nome em /proc.
func processRunning(comm string) bool {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err == nil && strings.TrimSpace(string(b)) == comm {
			return true
		}
	}
	return false
}

func tail(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines
}
