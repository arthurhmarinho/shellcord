# ADR-0016: Nível "Claude Code esperando você"

- Status: Aceito
- Data: 2026-11-14
- Substitui: ADR-0004

## Contexto

O ADR-0004 só tinha "Claude Code trabalhando". Com o Claude parado, a fonte dele não
aparecia. Além disso, o fish é pai do `claude` e reporta "comando rodando: claude"
enquanto a sessão está aberta. Esse estado ficava acima do Claude parado, e o card
mostrava "No terminal" sem dizer nada sobre o Claude.

## Decisão

Nova ordem de prioridade, da maior para a menor:

1. Claude Code trabalhando;
2. comando rodando no terminal;
3. **Claude Code esperando você** (sessão aberta, parada ou pedindo permissão);
4. shell parado;
5. nada.

- "Esperando você" tem o mesmo limite de inatividade do shell parado (10 min).
- A fonte do fish cujo comando é `claude` é ignorada: quem representa aquele terminal é
  a fonte do Claude Code.

## Consequências

- O card mostra a verdade quando o Claude espera, e um comando em outro terminal
  continua tendo prioridade.
- Custo: um `claude` sem os hooks instalados não aparece em lugar nenhum. A fonte do
  fish é ignorada e não existe fonte do Claude.
- Custo: a regra depende do nome do programa (`claude`). Um alias com outro nome volta
  a aparecer como comando rodando.

## Alternativas descartadas

- Deixar como estava: com o Claude parado, o card dizia "No terminal".
- Colocar "esperando você" acima de "comando rodando": um `cargo build` em outro
  terminal ficaria escondido atrás de uma sessão parada.

## Quando revisar

- Se você usar o Claude Code sem os hooks e sentir falta dele no card.
- Se você rodar o Claude Code por um alias ou wrapper com outro nome.
