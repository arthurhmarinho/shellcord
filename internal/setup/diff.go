package setup

import (
	"fmt"
	"strings"
)

// Diff mostra as linhas removidas (-) e acrescentadas (+), com contexto de até
// três linhas, a partir da maior subsequência comum. Os arquivos de config são
// pequenos, então a tabela O(n·m) não pesa.
func Diff(old, new string) string {
	a, b := splitLines(old), splitLines(new)
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type line struct {
		op   byte
		text string
	}
	var ops []line
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			ops = append(ops, line{' ', a[i]})
			i++
			j++
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, line{'-', a[i]})
			i++
		default:
			ops = append(ops, line{'+', b[j]})
			j++
		}
	}
	const ctx = 3
	show := make([]bool, len(ops))
	for k, o := range ops {
		if o.op != ' ' {
			for d := max(0, k-ctx); d <= min(len(ops)-1, k+ctx); d++ {
				show[d] = true
			}
		}
	}
	var sb strings.Builder
	skipped := false
	for k, o := range ops {
		if !show[k] {
			skipped = true
			continue
		}
		if skipped && sb.Len() > 0 {
			sb.WriteString("  …\n")
		}
		skipped = false
		fmt.Fprintf(&sb, "%c %s\n", o.op, o.text)
	}
	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
