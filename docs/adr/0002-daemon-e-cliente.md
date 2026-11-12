# ADR-0002: Daemon residente e cliente curto no mesmo binário

- Status: Aceito
- Data: 2026-11-11

## Contexto

A presença no Discord some quando a conexão IPC fecha. Um processo de vida curta, como o
hook do fish ou do Claude Code, não consegue manter a presença. Além disso, duas fontes
(fish e Claude Code, com várias sessões) precisam alimentar um único estado.

## Decisão

- Um daemon (`shellcord daemon`) segura a conexão com o Discord e o estado agregado.
- Um subcomando cliente curto (ex.: `shellcord send ...`) manda eventos ao daemon por um
  socket Unix em `$XDG_RUNTIME_DIR`.
- Um binário só, com subcomandos.

## Consequências

- Uma única conexão com o Discord e um único lugar para prioridade e limite de taxa.
- Instalação simples: um arquivo.
- Custo: há dois protocolos para manter, o do Discord e o nosso de eventos.
- Custo: o cliente precisa lidar com o daemon ausente (ver ADR-0006) sem atrasar o
  prompt. Um evento enviado com o daemon fora do ar pode se perder.

## Alternativas descartadas

- Dois binários (daemon e cliente): duplica build e instalação sem ganho real.
- O próprio fish segurando a conexão: não serve para o Claude Code e morre com o
  terminal.
- Comunicação por arquivo de estado em vez de socket: precisaria de polling ou inotify, e
  as escritas concorrentes de várias sessões disputariam o arquivo.

## Quando revisar

- Se o cliente passar de 10 ms por chamada medido no prompt do fish. Nesse caso, avaliar
  um envio sem esperar resposta ou um cliente separado e mínimo.
- Se surgir uma terceira fonte de eventos que não consiga executar um binário (ex.:
  extensão de navegador).

## Medições

- 2026-11-12, fase 2, loop de 300 chamadas no zsh com CPU em `powersave`:
  `shellcord send` leva 3,5 a 3,7 ms, contra 2,2 ms de um binário Go vazio e de
  `/bin/true`. O custo próprio é de uns 1,4 ms. Na primeira rodada, com a CPU fria,
  tudo levou de 3 a 4 vezes mais (`send` chegou a 10,5 ms). Repetir a medição dentro do
  fish na fase 4.
