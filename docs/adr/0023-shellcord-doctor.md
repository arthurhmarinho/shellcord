# ADR-0023: `shellcord doctor` só com fatos externos

- Status: Aceito
- Data: 2026-11-14

## Contexto

Quando o card não aparece, a causa pode estar em muitos lugares: Discord fechado,
daemon parado, config inválida (falha fechada, ADR-0020), hook faltando, `swayidle`
parado.

## Decisão

O `doctor` confere, sem perguntar nada ao daemon:

- o binário e a versão, e se o `shellcord` do PATH é o mesmo;
- o Discord, com um handshake real;
- o daemon vivo, testando o lock dele;
- a config, com o mesmo `config.Load` do daemon;
- as três integrações (`setup.Installed`) e o `swayidle` rodando;
- as últimas linhas do `daemon.log`.

Ele sai com código 1 se encontrar algum problema.

## Consequências

- Nenhuma mudança no daemon nem no protocolo.
- Custo: não mostra as fontes que o daemon conhece, só o último card no log.
- Custo: o handshake abre uma segunda conexão com o Discord por um instante.

## Alternativas descartadas

- Consultar o daemon por um `ctl.sock` (previsto no ADR-0009) com um `shellcord status`:
  diagnóstico completo, mas exige um segundo socket e um protocolo de consulta.

## Quando revisar

- Se um problema real não puder ser explicado pelo `doctor` e pelo log: aí entra o
  `status` pelo `ctl.sock`.
