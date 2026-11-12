# ADR-0008: Testes exploratórios fora do repositório

- Status: Aceito
- Data: 2026-11-11

## Contexto

A fase 1 é um teste descartável para validar o maior risco (sandbox do Flatpak) antes
de qualquer arquitetura.

## Decisão

Testes exploratórios e protótipos ficam em `~/Documentos/shellcord-testes/`, um
módulo Go por teste (ex.: `fase1-hello/`), fora deste repositório. Testes automatizados
(`go test`) do código de produção ficam no repositório, ao lado do código.

## Consequências

- O histórico do repositório começa só com código de produção.
- Custo: o protocolo IPC da fase 1 é copiado, não extraído, para `internal/ipc` na
  fase 2.
- Custo: os testes exploratórios não têm versionamento nem backup pelo git.

## Alternativas descartadas

- `cmd/hello/` no repositório: deixaria uma ferramenta de diagnóstico versionada, mas
  colocaria código descartável no histórico.

## Quando revisar

- Se você precisar de um teste exploratório pela segunda vez depois de apagado, ou
  quiser compartilhá-lo. Aí ele vira um subcomando de diagnóstico (ex.:
  `shellcord doctor`).
