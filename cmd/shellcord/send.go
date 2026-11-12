package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/arthurhmarinho/shellcord/internal/daemon"
	"github.com/arthurhmarinho/shellcord/internal/event"
	"github.com/arthurhmarinho/shellcord/internal/proc"
)

// claudeComm é o nome do processo do Claude Code em /proc/<pid>/stat
// (verificado com a versão 2.1.322, ADR-0010).
const claudeComm = "claude"

func runSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	src := fs.String("src", "", "fonte: fish ou claude")
	kind := fs.String("kind", "", "tipo: exec, idle, busy ou end")
	prog := fs.String("prog", "", "linha de comando; só o nome do programa é enviado")
	cwd := fs.String("cwd", "", "pasta da fonte (padrão: a atual)")
	id := fs.String("id", "", "session_id do Claude Code")
	pid := fs.Int("pid", 0, "PID da fonte (padrão: fish = processo pai; claude = ancestral \"claude\")")
	if err := fs.Parse(args); err != nil {
		return err
	}

	e := event.Event{
		V: event.Version, Src: event.Source(*src), Kind: event.Kind(*kind),
		ID: *id, Prog: event.ProgName(*prog), Cwd: *cwd,
	}
	if e.Cwd == "" {
		e.Cwd, _ = os.Getwd()
	}
	fillPID(&e, *pid)
	b, err := event.Encode(e)
	if err != nil {
		return err
	}
	return send(b)
}

// fillPID descobre o processo que representa a fonte (ADR-0010). Para o
// Claude Code, se a subida pela árvore falhar, o evento vai sem PID e a fonte
// expira por TTL no daemon.
func fillPID(e *event.Event, pid int) {
	switch {
	case pid > 0:
		e.PID = pid
	case e.Src == event.SourceFish:
		e.PID = os.Getppid()
	case e.Src == event.SourceClaude:
		if st, err := proc.FindAncestor(os.Getppid(), claudeComm, 8); err == nil {
			e.PID, e.Start = st.PID, st.Start
		}
		return
	}
	if st, err := proc.Read(e.PID); err == nil {
		e.Start = st.Start
	}
}

// send entrega o pacote sem esperar resposta (ADR-0009). Se o daemon não
// estiver rodando, sobe ele e descarta este evento (ADR-0006).
func send(b []byte) error {
	addr := &net.UnixAddr{Name: daemon.SocketPath(daemon.RuntimeDir()), Net: "unixgram"}
	c, err := net.DialUnix("unixgram", nil, addr)
	if err == nil {
		_, err = c.Write(b)
		c.Close()
	}
	if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
		return startDaemon()
	}
	return err
}

// startDaemon sobe "shellcord daemon" numa sessão nova, com a saída em
// daemon.log, e não espera.
func startDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dir := daemon.RuntimeDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "daemon.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "daemon")
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
