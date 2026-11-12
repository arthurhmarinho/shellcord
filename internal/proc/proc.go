// Package proc lê informações de processos no /proc (Linux), usadas para saber
// se uma fonte de eventos ainda está viva (ADR-0010).
package proc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// root é a raiz do procfs; os testes trocam por um diretório falso.
var root = "/proc"

// ErrNotFound indica que a subida pela árvore não achou o ancestral procurado.
var ErrNotFound = errors.New("ancestral não encontrado")

// Stat é o subconjunto de /proc/<pid>/stat que interessa ao shellcord.
type Stat struct {
	PID   int
	PPID  int
	Comm  string
	State byte
	Start uint64 // starttime: ticks desde o boot, campo 22
}

// Read lê /proc/<pid>/stat.
func Read(pid int) (Stat, error) {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "stat"))
	if err != nil {
		return Stat{}, err
	}
	return parse(pid, string(b))
}

// parse interpreta o conteúdo de /proc/<pid>/stat. O comm vem entre parênteses
// e pode conter espaços e parênteses, então os campos seguintes são contados a
// partir do último ")".
func parse(pid int, s string) (Stat, error) {
	open := strings.IndexByte(s, '(')
	end := strings.LastIndexByte(s, ')')
	if open < 0 || end < open {
		return Stat{}, fmt.Errorf("stat do pid %d malformado", pid)
	}
	f := strings.Fields(s[end+1:])
	// f[0] é o campo 3 (estado), f[1] é o 4 (ppid) e f[19] é o 22 (starttime).
	if len(f) < 20 || len(f[0]) != 1 {
		return Stat{}, fmt.Errorf("stat do pid %d com poucos campos", pid)
	}
	ppid, err := strconv.Atoi(f[1])
	if err != nil {
		return Stat{}, fmt.Errorf("ppid do pid %d: %w", pid, err)
	}
	start, err := strconv.ParseUint(f[19], 10, 64)
	if err != nil {
		return Stat{}, fmt.Errorf("starttime do pid %d: %w", pid, err)
	}
	return Stat{PID: pid, PPID: ppid, Comm: s[open+1 : end], State: f[0][0], Start: start}, nil
}

// Alive diz se o processo existe, não é zumbi e tem o mesmo starttime. Com
// start igual a zero, confere só a existência.
func Alive(pid int, start uint64) bool {
	st, err := Read(pid)
	if err != nil || st.State == 'Z' {
		return false
	}
	return start == 0 || st.Start == start
}

// FindAncestor sobe pela árvore a partir de pid (inclusive) até achar um
// processo com o comm dado, olhando no máximo maxDepth processos.
func FindAncestor(pid int, comm string, maxDepth int) (Stat, error) {
	for i := 0; i < maxDepth && pid > 1; i++ {
		st, err := Read(pid)
		if err != nil {
			return Stat{}, err
		}
		if st.Comm == comm {
			return st, nil
		}
		pid = st.PPID
	}
	return Stat{}, ErrNotFound
}
