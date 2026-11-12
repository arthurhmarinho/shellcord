# ADR-0007: Descoberta do socket do Discord, inclusive Flatpak

- Status: Aceito
- Data: 2026-11-11

## Contexto

O Discord está instalado como Flatpak (`com.discordapp.Discord`). Sem systemd-tmpfiles,
não existe o symlink usual para `$XDG_RUNTIME_DIR/discord-ipc-0`.

O que foi verificado em 2026-11-11, com o Discord 1.0.159 aberto:

- `/run/user/1000/app/com.discordapp.Discord/discord-ipc-0` existe;
  `/run/user/1000/discord-ipc-*` não existe.
- Esse socket é criado pelo wrapper do Flatpak, que roda
  `socat UNIX-LISTEN:<app>/discord-ipc-0,forever,fork UNIX-CONNECT:/run/user/1000/discord-ipc-0`
  dentro da sandbox. Ou seja, há um proxy entre nós e o Discord.
- O handshake e o `SET_ACTIVITY` funcionam por esse caminho (teste da fase 1).

## Decisão

- Procurar `discord-ipc-0` a `discord-ipc-9` em `$XDG_RUNTIME_DIR` e depois em
  `$XDG_RUNTIME_DIR/app/com.discordapp.Discord`.
- Considerar o Discord disponível só se `connect` e o handshake derem certo. O arquivo
  existir não basta, porque o socket do socat pode sobrar sem ninguém escutando.
- Timeout de leitura no handshake: com o socat no meio, um Discord travado pode aceitar
  a conexão e nunca responder.

## Consequências

- Funciona tanto com Discord nativo quanto Flatpak, sem configuração.
- Custo: até 20 tentativas de `connect` por rodada de reconexão. Em sockets Unix
  locais, isso custa microssegundos.

## Alternativas descartadas

- Caminho fixo na config: quebra se você trocar de instalação.
- Criar o symlink no arranque do sway: resolve só a máquina atual e esconde o problema.

## Quando revisar

- Se o wrapper do Flatpak do Discord mudar o caminho ou parar de usar o socat (conferir
  com `pgrep -af socat` com o Discord aberto).
- Se você instalar outra variante (Canary, PTB, Vesktop): os IDs do Flatpak mudam e
  podem precisar entrar na lista.
