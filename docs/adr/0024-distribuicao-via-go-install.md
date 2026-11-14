# ADR-0024: Distribuição via `go install`

- Status: Aceito
- Data: 2026-11-14

## Contexto

O README prometia `go install github.com/arthurhmarinho/shellcord@latest`, mas o
`main` fica em `cmd/shellcord`, então esse comando não funciona.

## Decisão

- A instalação é `go install github.com/arthurhmarinho/shellcord/cmd/shellcord@latest`.
  O layout com `cmd/` continua.
- `shellcord version` lê a versão que o próprio Go embute: a tag do módulo num
  `go install …@vX`, ou a pseudo-versão do git num build local (com `+dirty` se houver
  mudanças não commitadas).
- Sem binários prontos por enquanto.

## Consequências

- Nenhuma infraestrutura nova (sem CI, sem GoReleaser).
- Custo: quem instala precisa ter Go 1.27 ou mais novo.
- Custo: o comando de instalação é mais longo que o prometido antes.

## Alternativas descartadas

- Mover o `main` para a raiz: o comando ficaria curto, mas o `main.go` se misturaria
  com `go.mod` e `docs/`.
- Binários no GitHub Releases (GoReleaser + Actions): permitiria instalar sem Go, mas é
  CI para manter cedo demais num projeto com um usuário.

## Quando revisar

- Quando alguém pedir para instalar sem Go, ou quando houver a primeira tag de versão.
