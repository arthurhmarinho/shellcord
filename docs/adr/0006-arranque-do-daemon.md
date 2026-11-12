# ADR-0006: Arranque do daemon: sway e autoarranque pelo cliente

- Status: Aceito
- Data: 2026-11-11

## Contexto

A máquina usa Void Linux com runit e sem systemd, então não há unidades de usuário. O
daemon precisa estar de pé quando o primeiro evento chega.

## Decisão

As duas formas ao mesmo tempo:

- `exec shellcord daemon` no `~/.config/sway/config`;
- o cliente, ao não conseguir conectar no socket local, sobe o daemon desacoplado
  (sessão nova) e segue sem esperar.

## Consequências

- Funciona mesmo fora do sway ou depois de o daemon cair.
- Custo: risco de dois daemons ao mesmo tempo (sway e cliente em paralelo). Um lock
  exclusivo (ex.: `flock` em um arquivo em `$XDG_RUNTIME_DIR`) é obrigatório.
- Custo: o evento que disparou o autoarranque pode se perder, o que é aceitável.

## Alternativas descartadas

- Serviço de usuário do runit (`runsvdir` por usuário): mais robusto, mas exige
  configuração fora do home e root para instalar.
- Só o autoarranque: o primeiro comando do dia pagaria o arranque e o primeiro evento
  se perderia sempre.

## Quando revisar

- Se o daemon cair com frequência a ponto de precisar de supervisão de verdade. Aí o
  serviço de usuário do runit volta à mesa.
- Se você trocar de compositor ou passar a usar o shellcord fora do sway.
