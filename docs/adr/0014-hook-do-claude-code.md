# ADR-0014: Subcomando `shellcord hook claude` para os hooks do Claude Code

- Status: Aceito
- Data: 2026-11-14

## Contexto

O Claude Code chama cada hook com um JSON no stdin: `session_id`, `cwd`,
`hook_event_name` e, dependendo do hook, `tool_name`. Esse JSON também traz conteúdo
sensível: o texto do prompt (`UserPromptSubmit`), a entrada da ferramenta
(`tool_input`, ex.: o comando inteiro), a saída dela (`tool_response`) e a última
mensagem do assistente (`Stop`). A saída do `SessionStart` e do `UserPromptSubmit` é
injetada no contexto do Claude, e um código de saída 2 bloqueia a ação.

## Decisão

- A mesma linha de comando em todos os hooks: `shellcord hook claude`. O mapeamento de
  hook para evento fica no código:

  | Hook | Evento |
  |---|---|
  | SessionStart | `idle` |
  | UserPromptSubmit, PreToolUse, PostToolUse | `busy` (com `tool_name`, se houver) |
  | Notification | `idle`: o Claude está esperando você (permissão ou input) |
  | Stop | `idle` |
  | SessionEnd | `end` |
  | qualquer outro | ignorado |

- Só `session_id`, `cwd`, `hook_event_name` e `tool_name` são decodificados. O resto do
  stdin é descartado sem ser interpretado.
- O subcomando **nunca escreve no stdout** e **sempre sai com código 0**, inclusive em
  caso de pânico. Os erros só aparecem no stderr com `SHELLCORD_DEBUG=1`.

## Consequências

- O `settings.json` fica com uma linha repetida, e a lógica fica testável em Go.
- Um defeito no shellcord não interrompe nem polui a sessão do Claude Code.
- Custo: os hooks são síncronos. Cada um atrasa a ferramenta em uns 6 ms, medidos em
  2026-11-14.
- Custo: os erros ficam invisíveis por padrão. Para depurar, é preciso rodar com
  `SHELLCORD_DEBUG=1`.

## Alternativas descartadas

- `shellcord send` com `jq` em cada hook: depende do `jq` e espalha 7 linhas diferentes
  pelo `settings.json`.

## Quando revisar

- Se o Claude Code mudar os nomes dos hooks ou os campos do JSON (os testes em
  `cmd/shellcord/hook_test.go` usam payloads capturados com a versão 2.1.322).
- Se o Claude Code passar a oferecer hooks assíncronos: aí o atraso de 6 ms some.
