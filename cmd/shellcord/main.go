// Comando shellcord: Discord Rich Presence para o terminal e o Claude Code.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/arthurhmarinho/shellcord/internal/daemon"
)

// defaultClientID é a aplicação "shellcord" no Discord Developer Portal. Não é
// segredo: o Discord o mostra em toda atividade (ADR-0013).
const defaultClientID = "1570271223571812512"

const usage = `uso: shellcord <comando> [opções]

comandos:
  install     instala as integrações (Claude Code, fish, sway), com diff antes
  uninstall   remove as integrações, com diff antes
  doctor      confere Discord, daemon, config e integrações
  config      valida e mostra a config: "shellcord config check"
  version     mostra a versão

  daemon      mantém a conexão com o Discord e publica a presença
  send        manda um evento ao daemon (sobe o daemon se precisar)
  hook        usado pelos hooks do Claude Code: "shellcord hook claude"
  init        imprime a integração com o shell: "shellcord init fish | source"
  away        avisa que você saiu (para o swayidle); o card some
  back        avisa que você voltou (para o swayidle)

Use "shellcord <comando> -h" para ver as opções.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "daemon":
		err = runDaemon(os.Args[2:])
	case "send":
		err = runSend(os.Args[2:])
	case "hook":
		runHook(os.Args[2:]) // nunca retorna: sai sempre com 0
	case "init":
		err = runInit(os.Args[2:])
	case "config":
		err = runConfig(os.Args[2:])
	case "install", "uninstall":
		err = runInstall(os.Args[1], os.Args[2:])
	case "doctor":
		err = runDoctor(os.Args[2:])
	case "version", "-version", "--version":
		fmt.Println("shellcord", version())
		return
	case "away", "back":
		err = runPresence(os.Args[1], os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "shellcord: comando desconhecido %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "shellcord:", err)
		os.Exit(1)
	}
}

func runDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := os.Getenv("SHELLCORD_CLIENT_ID")
	if id == "" {
		id = defaultClientID
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = daemon.Run(ctx, daemon.Config{ClientID: id, Executable: exe})
	stop()
	if errors.Is(err, daemon.ErrRestart) {
		// O Run já soltou o lock, o socket e a conexão: troca a imagem do
		// processo pelo executável novo, com os mesmos argumentos (ADR-0021).
		return syscall.Exec(exe, os.Args, os.Environ())
	}
	if errors.Is(err, daemon.ErrRunning) {
		// Normal quando o sway e o autoarranque disputam: sai sem erro.
		fmt.Fprintln(os.Stderr, "shellcord:", err)
		return nil
	}
	return err
}
