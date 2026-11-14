# ADR-0018: Desembrulhar wrappers no nome do programa

- Status: Aceito
- Data: 2026-11-14

## Contexto

O card mostra só o nome do programa (ADR-0005). Com a primeira palavra,
`sudo nvim /etc/fstab` vira `sudo` e `env FOO=1 cargo build` vira `env`, e o card perde
o que importa.

## Decisão

`event.ProgName` pula:

- atribuições `VAR=valor` em qualquer posição antes do programa;
- uma lista fixa de wrappers: `sudo`, `doas`, `env`, `time`, `nice`, `command`,
  `builtin`, `exec` e `nohup`;
- as opções desses wrappers. As opções de uma letra que levam valor (ex.: `sudo -u
  root`, `nice -n 10`) pulam também a palavra seguinte, e `--` encerra as opções.

A primeira palavra que sobra é o programa, sem caminho. Se só houver wrappers, o
resultado é o último deles; se só houver atribuições, não há programa.

## Consequências

- "Rodando nvim" em vez de "sudo".
- Errar aqui não vaza nada: qualquer nome passa pela allowlist depois. Um caso que
  escapa da lista aparece como "No terminal".
- Custo: a lista nunca fica completa (`sudo --user=x` funciona, `sudo --user x` não),
  e cada wrapper novo precisa de teste.
- Custo: o `sudo` some do card. Isso não é um problema, porque o card nunca teve a
  intenção de mostrar privilégios.

## Alternativas descartadas

- Primeira palavra: nada a manter, mas `sudo` fora da allowlist apagaria a informação.
- Interpretar a linha com um parser de shell: correto em mais casos, mas pesado para
  um ganho pequeno, porque a allowlist já protege a privacidade.

## Quando revisar

- Se um comando que você usa com frequência aparecer com o nome do wrapper.
- Se aparecer um wrapper comum fora da lista (ex.: `systemd-run`, `flatpak run`).
