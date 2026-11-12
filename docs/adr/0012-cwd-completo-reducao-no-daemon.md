# ADR-0012: Caminho completo no evento, redução no daemon

- Status: Aceito
- Data: 2026-11-12

## Contexto

O card mostra o nome da pasta (ADR-0005), e a lista de pastas ignoradas (fase 5)
precisa decidir o que esconder. O cliente pode mandar o caminho completo ou só o
último componente.

## Decisão

O evento leva o `cwd` completo. O daemon reduz para o último componente (a home vira
`~`) só ao montar o card, em `internal/daemon/render.go`. Nenhum outro ponto do código
manda caminho para o Discord.

## Consequências

- A lista de pastas ignoradas pode casar caminhos completos e prefixos: ignorar
  `~/Documentos/clientes` esconde também as subpastas.
- Custo: o caminho completo trafega no socket local. Ele fica restrito ao seu usuário,
  porque o diretório `$XDG_RUNTIME_DIR/shellcord` tem modo 0700.
- Custo: a garantia de privacidade depende de um único ponto (`render`), que precisa de
  teste cobrindo a redução.

## Alternativas descartadas

- Só o último componente no evento: menos dado trafegando, mas a lista de pastas
  ignoradas só casaria por nome e não pegaria as subpastas.

## Quando revisar

- Se algum campo do card passar a ser montado fora de `render`.
- Se o socket local passar a aceitar outros usuários (ex.: modo multiusuário).
