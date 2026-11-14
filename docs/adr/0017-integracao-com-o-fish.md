# ADR-0017: Integração com o fish via `shellcord init fish`

- Status: Aceito
- Data: 2026-11-14

## Contexto

O fish emite `fish_preexec` (com a linha de comando em `$argv[1]`), `fish_postexec` e
`fish_exit`. O cliente precisa ser chamado nesses eventos sem atrasar o prompt de forma
perceptível e sem vazar a linha de comando.

## Decisão

- O `config.fish` carrega a integração com `shellcord init fish | source`, dentro do
  `if status is-interactive`, igual ao starship. O script é gerado pelo binário e
  embute o caminho absoluto dele, então não depende do PATH.
- Os eventos: ao carregar, `idle`; em `fish_preexec`, `exec`; em `fish_postexec`,
  `idle` (com o `$PWD` novo, que pega o `cd`); em `fish_exit`, `end`. O PID é o
  `$fish_pid`.
- **Envio síncrono**: o handler espera o `send` terminar (uns 2 ms).
- **A linha de comando vai pela variável de ambiente `SHELLCORD_CMDLINE`, não por
  argumento.** O `/proc/<pid>/cmdline` pode ser lido por qualquer usuário da máquina;
  o `/proc/<pid>/environ` só pelo dono. O cliente reduz a linha ao nome do programa
  antes de enviar.
- Uma linha só com atribuições (`FOO=1`) não gera evento.

## Consequências

- Atualizar o binário atualiza o script no próximo terminal aberto.
- Os eventos chegam em ordem: o `idle` de um comando rápido nunca passa na frente do
  `exec` dele.
- Custo: uns 3,5 ms ao abrir cada terminal (o `init`) e uns 2 ms no início e no fim de
  cada comando (ADR-0002, medições).
- Custo: a integração depende dos nomes dos eventos do fish.

## Alternativas descartadas

- Arquivo em `~/.config/fish/conf.d/`: custo zero ao abrir o terminal, mas fica velho
  quando o binário muda.
- Envio em segundo plano (`&` + `disown`): o prompt não esperaria, mas o `idle` de um
  comando rápido poderia chegar antes do `exec`, deixando o card preso em "Rodando ls".
  Ainda haveria as mensagens de job do fish.
- Linha de comando por argumento (`-prog "$argv[1]"`): ficaria visível para outros
  usuários em `/proc` enquanto o cliente roda.

## Quando revisar

- Se o fish mudar ou remover `fish_preexec`/`fish_postexec`.
- Se outro shell (bash, zsh) entrar no projeto: aí o `init` ganha um argumento por shell.
- Se a latência medida dentro do fish passar de 10 ms por comando.
