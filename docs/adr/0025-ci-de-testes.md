# ADR-0025: CI só de testes no GitHub Actions

- Status: Aceito
- Data: 2026-11-15

## Contexto

O repositório é público. Os testes cobrem o que mais importa (privacidade no `render`,
falha fechada da config, edição do `settings.json`), mas só rodam quando alguém lembra
de rodar `go test` localmente.

## Decisão

Um workflow (`.github/workflows/testes.yml`) roda a cada push e pull request, num
runner Linux (os testes dependem de `/proc` e de sockets Unix):

- `gofmt -l` (falha se algum arquivo estiver fora do formato);
- `go vet ./...`;
- `go test -race -count=1 ./...`.

A versão do Go vem do `go.mod`. As actions são `actions/checkout@v7` e
`actions/setup-go@v7`, fixadas pela versão maior. O workflow só tem permissão de leitura
do repositório.

## Consequências

- Uma regressão aparece no push, inclusive em pull requests de outras pessoas.
- Custo: um arquivo de workflow para manter, e as actions precisam ser atualizadas
  quando sair uma versão maior nova.
- Custo: fixar pela versão maior (`@v7`) aceita atualizações menores sem revisão. Fixar
  pelo hash do commit seria mais seguro contra uma action comprometida, mas daria mais
  trabalho para atualizar.

## Alternativas descartadas

- Sem CI: nada a manter, mas os testes dependeriam de memória.
- CI com binários prontos (GoReleaser): continua descartado pelo ADR-0024.

## Quando revisar

- Se o projeto passar a gerar binários: aí entra o GoReleaser.
- Se surgir notícia de uma action comprometida: aí as actions passam a ser fixadas pelo
  hash do commit.
