# ADR-0005: Privacidade por padrão e config em TOML

- Status: Aceito
- Data: 2026-11-11

## Contexto

A presença é pública para quem vê o perfil. Os argumentos de comandos podem conter
caminhos, hosts, tokens e nomes de clientes. Os nomes de pasta também podem ser
sensíveis.

## Decisão

- Mostrar só o nome do programa, nunca os argumentos. Os argumentos não chegam nem ao
  daemon: o cliente corta antes de enviar.
- Allowlist de programas: fora dela aparece algo genérico ("no terminal").
- O nome da pasta aparece (só o último componente), com uma lista de pastas ignoradas.
- Config em `~/.config/shellcord/config.toml`. Sem config, valem os padrões acima.

## Consequências

- Vazamento acidental exige mudar o código, não só esquecer uma linha na config.
- Custo: uma dependência externa para ler TOML (a biblioteca padrão do Go não tem), ou
  um parser mínimo próprio. A escolha fica para a fase 5.
- Custo: programas fora da allowlist aparecem como genéricos até você adicioná-los.

## Alternativas descartadas

- Denylist de programas: qualquer programa esquecido vaza.
- Mostrar a linha de comando inteira, como algumas extensões de IDE: vaza segredos.
- Config em JSON: sem comentários, pior de editar à mão.

## Quando revisar

- Se aparecer no card qualquer coisa que você não quis expor. Nesse caso, o padrão
  correspondente fica mais restritivo.
- Se a dependência de TOML escolhida ficar sem manutenção por 12 meses.
