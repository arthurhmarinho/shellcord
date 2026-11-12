package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeStat monta uma linha de /proc/<pid>/stat com os campos que interessam.
func fakeStat(pid, ppid int, comm string, state byte, start uint64) string {
	f := make([]string, 50)
	for i := range f {
		f[i] = "0"
	}
	f[0] = string(state)
	f[1] = fmt.Sprint(ppid)
	f[19] = fmt.Sprint(start)
	return fmt.Sprintf("%d (%s) %s\n", pid, comm, strings.Join(f, " "))
}

func withFakeProc(t *testing.T, procs map[int]string) {
	t.Helper()
	dir := t.TempDir()
	for pid, content := range procs {
		p := filepath.Join(dir, fmt.Sprint(pid))
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "stat"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := root
	root = dir
	t.Cleanup(func() { root = old })
}

func TestParseCommWithParens(t *testing.T) {
	st, err := parse(10, fakeStat(10, 3, "a) (b", 'S', 999))
	if err != nil {
		t.Fatal(err)
	}
	if st.Comm != "a) (b" || st.PPID != 3 || st.Start != 999 || st.State != 'S' {
		t.Fatalf("parse errado: %+v", st)
	}
}

func TestReadSelf(t *testing.T) {
	st, err := Read(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if st.PPID != os.Getppid() || st.Start == 0 {
		t.Fatalf("stat do próprio processo inesperado: %+v", st)
	}
	if !Alive(st.PID, st.Start) {
		t.Fatal("o próprio processo deveria estar vivo")
	}
	if Alive(st.PID, st.Start+1) {
		t.Fatal("starttime diferente deveria contar como morto (PID reutilizado)")
	}
}

func TestFindAncestor(t *testing.T) {
	withFakeProc(t, map[int]string{
		100: fakeStat(100, 50, "shellcord", 'R', 5),
		50:  fakeStat(50, 20, "sh", 'S', 4),
		20:  fakeStat(20, 10, "claude", 'S', 3),
		10:  fakeStat(10, 1, "fish", 'S', 2),
	})
	st, err := FindAncestor(100, "claude", 8)
	if err != nil || st.PID != 20 || st.Start != 3 {
		t.Fatalf("FindAncestor = %+v, %v", st, err)
	}
	if _, err := FindAncestor(100, "claude", 2); err != ErrNotFound {
		t.Fatalf("com profundidade 2 não deveria achar: %v", err)
	}
	if _, err := FindAncestor(100, "vim", 8); err != ErrNotFound {
		t.Fatalf("comm inexistente: %v", err)
	}
}

func TestAliveZombie(t *testing.T) {
	withFakeProc(t, map[int]string{7: fakeStat(7, 1, "x", 'Z', 1)})
	if Alive(7, 1) {
		t.Fatal("zumbi não está vivo")
	}
	if Alive(8, 0) {
		t.Fatal("pid inexistente não está vivo")
	}
}
