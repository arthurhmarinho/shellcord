# ADR-0003: Coalescência de atualizações, reconexão e limpeza por inatividade

- Status: Aceito
- Data: 2026-11-11

## Contexto

O Discord aceita cerca de 1 atualização de presença a cada 15 s. O fish gera eventos a
cada comando, e o Claude Code a cada ferramenta usada, ou seja, muito mais que isso. O
Discord também pode fechar ou reabrir a qualquer momento.

## Decisão

- O daemon guarda só o estado mais recente. Quando a janela de 15 s libera, ele manda
  esse estado se for diferente do último enviado. Estados intermediários são descartados.
- Reconexão: se a conexão cair, o daemon tenta de novo periodicamente (com backoff) e,
  ao reconectar, reenvia o estado atual.
- Limpeza: depois de N minutos sem eventos, a presença é limpa (`SET_ACTIVITY` sem
  `activity`). O valor de N vai para a config (ADR-0005).

## Consequências

- Nunca estoura o limite e o Discord sempre mostra o estado mais atual possível.
- Custo: até 15 s de atraso entre um evento e o que aparece no perfil. Comandos curtos
  podem nunca aparecer, o que é aceitável.
- Custo: o daemon precisa de um relógio e de uma máquina de estados simples, e esses
  testes dependem de tempo (por isso o relógio deve ser injetável).

## Alternativas descartadas

- Mandar todo evento e deixar o Discord descartar: o Discord pode limitar a aplicação e
  o resultado visível fica imprevisível.
- Fila de eventos: mostraria estados antigos com atraso crescente.

## Quando revisar

- Se o Discord mudar o limite (respostas de erro de rate limit no `SET_ACTIVITY`, ou um
  intervalo medido diferente de 15 s).
- Se o atraso de 15 s incomodar no uso real a ponto de você pedir uma atualização
  imediata para algum tipo de evento.
