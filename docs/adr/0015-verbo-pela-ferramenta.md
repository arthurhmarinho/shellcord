# ADR-0015: O card mostra um verbo derivado da ferramenta do Claude Code

- Status: Aceito
- Data: 2026-11-14

## Contexto

"Claude Code trabalhando" é estático. O `tool_name` dos hooks diz o que o Claude está
fazendo, mas o nome cru pode revelar demais: uma ferramenta MCP como
`mcp__gmail__send` mostra quais integrações você usa.

## Decisão

- O evento ganha o campo `tool` (aditivo, continua `v: 1`), preenchido só em `busy`.
- O daemon traduz o nome por uma tabela fixa em `render`:

  | Ferramentas | Card |
  |---|---|
  | Edit, MultiEdit, Write, NotebookEdit | editando arquivos |
  | Read, Grep, Glob | lendo código |
  | Bash | rodando comandos |
  | WebFetch, WebSearch | pesquisando na web |
  | qualquer outra, inclusive MCP | trabalhando |

- Trocar de ferramenta não reinicia o cronômetro, só mudar de nível. Com várias
  sessões, aparece o verbo da mais antiga (ADR-0011).

## Consequências

- O card acompanha o trabalho, limitado à janela de 15 s.
- O nome cru da ferramenta só trafega no socket local, como o `cwd` (ADR-0012).
- Custo: a tabela precisa acompanhar as ferramentas novas do Claude Code. Até alguém
  adicionar, uma ferramenta nova aparece como "trabalhando".
- Custo: o card pode mudar de texto a cada 15 s durante uma sessão longa.

## Alternativas descartadas

- Só "Claude Code trabalhando": nada a manter, mas o card fica parado.
- Mostrar o `tool_name` cru: expõe as integrações MCP.

## Quando revisar

- Se uma ferramenta que você usa muito aparecer como "trabalhando".
- Se a troca de texto a cada 15 s incomodar no uso real.
