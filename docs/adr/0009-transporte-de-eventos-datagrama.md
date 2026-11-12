# ADR-0009: Transporte de eventos por datagrama Unix

- Status: Aceito
- Data: 2026-11-12

## Contexto

O cliente roda nos hooks do fish (a cada comando) e do Claude Code (a cada ferramenta).
Ele precisa entregar um evento pequeno ao daemon (ADR-0002) e sair o mais rápido
possível, sem esperar o daemon processar.

## Decisão

- O daemon escuta um socket Unix `SOCK_DGRAM` em `$XDG_RUNTIME_DIR/shellcord/events.sock`.
  O diretório tem modo 0700, e o lock do daemon (ADR-0006) fica no mesmo diretório.
- Cada evento é um pacote com um objeto JSON e campo de versão (`"v": 1`).
- O cliente não espera resposta: monta o pacote, faz `sendto` e sai.
- Se o `sendto` falhar com `ENOENT` ou `ECONNREFUSED`, o daemon não está rodando, e esse
  é o sinal para o autoarranque (ADR-0006).

## Consequências

- O caminho quente é um `socket` e um `sendto`, sem `connect`, sem leitura e sem
  timeout. Cada pacote chega inteiro, sem precisar delimitar mensagens.
- O daemon não gerencia conexões nem clientes pendurados.
- Custo: não há confirmação de que o evento foi processado. Um pacote pode se perder se
  o buffer do socket encher, o que é improvável com o daemon vivo.
- Custo: comandos com resposta (ex.: um futuro `shellcord status`) precisam de outro
  canal: um socket stream `ctl.sock` no mesmo diretório.

## Alternativas descartadas

- Socket stream com JSON por linha: um socket só para eventos e consultas, e fácil de
  depurar com `socat`, mas o cliente precisa de `connect` e o daemon precisa gerenciar
  conexões e timeouts.
- Protocolo binário: ganho de desempenho irrelevante, porque o pacote tem menos de 1 KB,
  e fica impossível de inspecionar.

## Quando revisar

- Se aparecer evento perdido no uso real (ex.: o card não atualiza com o daemon vivo).
- Quando o primeiro comando com resposta for implementado: confirmar que o `ctl.sock`
  separado não duplica lógica demais. Se duplicar, avaliar migrar tudo para stream.
