<div align="center">

```
      _          _ _                     _
  ___| |__   ___| | | ___ ___  _ __ __| |
 / __| '_ \ / _ \ | |/ __/ _ \| '__/ _` |
 \__ \ | | |  __/ | | (_| (_) | | | (_| |
 |___/_| |_|\___|_|_|\___\___/|_|  \__,_|

        ><>  o seu terminal, ao vivo no Discord  ><>
```

**O VS Code tem Rich Presence. O Neovim tem. O seu prompt do fish, até agora, não tinha.**

</div>

---

## O que é isso

O **shellcord** mostra no seu perfil do Discord o que você está fazendo no terminal e
no [Claude Code](https://claude.com/claude-code). É o mesmo "Jogando …" que as IDEs
exibem, só que para quem vive no `❯`.

```
┌─────────────────────────────────────────────┐
│  JOGANDO                                    │
│  ┌───────┐                                  │
│  │       │   shellcord                      │
│  │  ><>  │   rodando cargo                  │
│  │       │   em shellcord                   │
│  └───────┘   há 12 minutos                  │
└─────────────────────────────────────────────┘
```

Quando o Claude Code está trabalhando, ele passa na frente:

```
┌─────────────────────────────────────────────┐
│  JOGANDO                                    │
│  ┌───────┐                                  │
│  │       │   shellcord                      │
│  │  ✻    │   Claude Code editando arquivos  │
│  │       │   em shellcord                   │
│  └───────┘   há 3 minutos                   │
└─────────────────────────────────────────────┘
```

E quando você vai tomar café, ele some sozinho. Com o `swayidle` avisando que ninguém
mexe no teclado há 10 minutos, o card sai do perfil, mesmo com um `htop` esquecido
aberto. Só o Claude Code trabalhando continua aparecendo, porque ele está mesmo
trabalhando.

## Como funciona

A presença do Discord existe enquanto alguém segura a conexão com ele, e um hook de
shell vive alguns milissegundos. Por isso o shellcord tem duas partes no mesmo binário:

```
 fish  (fish_preexec / fish_postexec) ──┐
                                        ├──► shellcord send … ──► socket local ──► shellcord daemon ──► Discord
 Claude Code (hooks, JSON no stdin) ────┘                                              │
                                                                       ┌───────────────┴───────────────┐
                                                                       │ prioridade entre as fontes    │
                                                                       │ uma atualização a cada 15 s   │
                                                                       │ reconexão quando o Discord    │
                                                                       │   fecha e reabre              │
                                                                       │ limpeza por inatividade       │
                                                                       └───────────────────────────────┘
```

- **O daemon** segura a conexão IPC com o Discord. É Go puro, com o protocolo escrito à
  mão: um cabeçalho de 8 bytes e um JSON, nada além disso.
- **O cliente** (`shellcord send`) é chamado pelos hooks, entrega o evento e sai.
  Roda a cada comando, então precisa ser rápido a ponto de você não perceber.
- **A ordem de prioridade** decide o que aparece:

  | | Situação | Exemplo |
  |---|---|---|
  | 1 | Claude Code trabalhando | *Claude Code editando arquivos* |
  | 2 | Comando rodando no terminal | *rodando cargo* |
  | 3 | Claude Code esperando você | *Claude Code esperando você* |
  | 4 | Shell parado | *no terminal* |
  | 5 | Parado há N minutos | *(card some)* |

## Privacidade, antes de tudo

Seu perfil é público. Seu histórico de comandos não deveria ser.

- **Só o nome do programa, nunca os argumentos.** `curl -H "Authorization: Bearer …"`
  aparece como `curl` e pronto. Os argumentos são cortados **no cliente**, antes de
  chegar ao daemon. Para vazar alguma coisa seria preciso mudar o código, não só
  esquecer uma linha da config.
- **Allowlist, não denylist.** Programa que você não liberou aparece como *no
  terminal*. Um programa esquecido não vaza nada.
- **Pastas ignoráveis.** Só aparece o último componente do caminho (`shellcord`, não
  `/home/você/clientes/projeto-secreto/shellcord`), e as pastas da lista de ignoradas
  não aparecem de jeito nenhum.

A config vai morar em `~/.config/shellcord/config.toml`. Um esboço do formato (ainda não
implementado, isso é a fase 5):

```toml
# Programas que podem aparecer pelo nome. O resto vira "no terminal".
allow = ["nvim", "cargo", "go", "git", "make", "htop"]

# Pastas que nunca aparecem (nem o nome, nem nada dentro delas).
ignore_dirs = ["~/Documentos/financeiro", "~/clientes"]

# Minutos com o shell parado no prompt até o card sumir.
idle_minutes = 10
```

## Discord Flatpak? Funciona

Se o seu Discord é Flatpak, o socket IPC não fica no lugar padrão: fica em
`$XDG_RUNTIME_DIR/app/com.discordapp.Discord/discord-ipc-0`, atrás de um `socat` que o
próprio wrapper do Flatpak sobe. A receita comum é um symlink criado pelo
systemd-tmpfiles. O shellcord procura nos dois lugares sozinho e só considera o
Discord disponível depois de um handshake bem-sucedido.

Isso foi testado num Void Linux com runit, **sem systemd**, então não há unidade de
usuário para instalar.

## Status

O projeto está no começo. Por fase:

- [x] **Fase 1:** prova de conceito. Handshake e `SET_ACTIVITY` pelo socket do Flatpak,
      com o card aparecendo no perfil.
- [x] **Fase 2:** daemon, com socket local, prioridade, espera de 15 s, reconexão e
      limpeza por inatividade. Já dá para mandar eventos à mão com `shellcord send`.
- [x] **Fase 3:** hooks do Claude Code, com várias sessões ao mesmo tempo
      (`shellcord hook claude`).
- [x] **Fase 4:** integração com o fish (`shellcord init fish | source`) e ausência
      via `swayidle` (`shellcord away` / `shellcord back`).
- [ ] **Fase 5:** config TOML com allowlist e pastas ignoradas.
- [ ] **Fase 6:** `exec` no sway, script de instalação e um `shellcord doctor`.

## Instalação

*Ainda não tem.* Quando a fase 6 chegar, a ideia é esta:

```fish
go install github.com/arthurhmarinho/shellcord@latest
shellcord install   # hooks do fish e do Claude Code, mostrando o diff antes
```

Mais uma linha no config do sway (o cliente também sobe o daemon sozinho se ele não
estiver rodando):

```
exec shellcord daemon
```

Não precisa criar nada no Discord Developer Portal: o binário já vem com o
**Application ID** da aplicação "shellcord", que é o nome que aparece depois de
"Jogando". Esse ID não é segredo, o próprio Discord o mostra publicamente. Para usar
uma aplicação sua, com outro nome no card, defina `SHELLCORD_CLIENT_ID`.

## Perguntas que alguém vai fazer

**Por que o card demora até 15 segundos para mudar?**
O Discord aceita mais ou menos uma atualização a cada 15 s. O daemon guarda só o
estado mais recente e manda quando a janela libera. Um `ls` de 40 ms nunca vai
aparecer, e é melhor assim.

**Por que não mostra o comando inteiro, igual algumas extensões de IDE?**
Por causa de `export AWS_SECRET_ACCESS_KEY=…`.

**E bash/zsh?**
O desenho não depende do fish: qualquer shell com `preexec` consegue chamar
`shellcord send`. O fish vem primeiro porque é o shell de quem escreve.

**Por que Go?**
Porque o cliente roda a cada Enter que você aperta. O arranque de um binário Go fica
em poucos milissegundos, e o do Node em dezenas.

## Por dentro

Cada decisão de arquitetura (o que foi decidido, quanto custa, o que foi descartado e
quando revisar) está registrada em [`docs/adr/`](docs/adr/README.md). Se alguma coisa
no código parecer estranha, a explicação provavelmente está lá.

---

<div align="center">

feito no terminal, para o terminal ><>

</div>
