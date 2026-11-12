# ADR-0001: Go e protocolo IPC do Discord escrito à mão

- Status: Aceito
- Data: 2026-11-11

## Contexto

O cliente do shellcord roda a cada comando do fish (preexec e postexec), então o tempo
de arranque aparece na latência do prompt. O daemon fica residente e segura a conexão
com o Discord. As bibliotecas Go de Discord RPC estão abandonadas.

O protocolo é pequeno: cada mensagem tem um cabeçalho de 8 bytes (opcode `uint32` e
tamanho `uint32`, ambos little-endian) seguido de JSON. Opcodes: 0 HANDSHAKE, 1 FRAME,
2 CLOSE, 3 PING, 4 PONG. A sequência é HANDSHAKE, esperar o `READY` e depois mandar
`SET_ACTIVITY` em FRAME.

## Decisão

Go, só com a biblioteca padrão para o IPC. O protocolo é implementado à mão em um
pacote interno (`internal/ipc`, a partir da fase 2). Module path:
`github.com/arthurhmarinho/shellcord`.

## Consequências

- Binário estático, com arranque na casa de poucos milissegundos e sem runtime externo.
- Nenhuma dependência abandonada no caminho crítico.
- Custo: o tratamento de PING/PONG, CLOSE e códigos de erro fica conosco. Se o Discord
  mudar o protocolo, ninguém corrige por nós.
- Custo: o binário tem uns 5 MB, contra poucos KB de um script, o que é irrelevante aqui.

## Alternativas descartadas

- Bibliotecas Go de RPC (ex.: rich-go): abandonadas e com APIs que escondem o CLOSE.
- Node: o arranque de ~50 ms por comando do fish pesa no prompt.
- Script em fish ou shell falando com o socket via `socat`: não tem como manter a conexão
  e coalescer atualizações sem um processo residente.

## Quando revisar

- Se o Discord publicar uma mudança no protocolo IPC (versão do handshake diferente de
  `v: 1`) ou o `SET_ACTIVITY` passar a ser recusado.
- Se surgir uma biblioteca Go de RPC mantida, com commits nos últimos 6 meses e suporte a
  reconexão, que substitua mais de 200 linhas nossas.
