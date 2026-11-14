# ADR-0020: Config TOML: leitura, releitura e falha fechada

- Status: Aceito
- Data: 2026-11-14

## Contexto

O ADR-0005 definiu a privacidade (allowlist e pastas ignoradas) e a config em TOML,
mas deixou para a fase 5 como ler o arquivo, quando relê-lo e o que fazer com um erro.
Só o daemon lê a config, então nada disso afeta a latência do cliente.

## Decisão

- **Leitor:** `github.com/BurntSushi/toml` v1.6.0 (dezembro de 2025), a primeira
  dependência externa do projeto. Chave desconhecida é erro.
- **Releitura:** o daemon confere a data de modificação e o tamanho do arquivo na
  checagem periódica que já existe (15 s) e relê quando mudam. Não usa inotify nem
  sinal.
- **Semântica:**
  - `allow` substitui a lista padrão. Uma lista vazia esconde todos os programas.
  - `ignore_dirs` aceita `~` e casa por prefixo de caminho, cobrindo as subpastas.
    Uma fonte numa pasta ignorada sai do cálculo do card por inteiro: não aparece nem
    entra na contagem.
  - `idle_minutes` precisa ser pelo menos 1.
- **Falha fechada:** com a config inválida ao subir, o daemon não publica nada até ela
  ser corrigida. Numa releitura inválida, mantém a última config válida. Os erros vão
  para o log do daemon, e `shellcord config check` valida o arquivo na hora.
- `shellcord config default` imprime a config padrão, a mesma usada sem arquivo (um
  teste garante isso).

## Consequências

- Um erro de digitação em `ignore_dirs` nunca faz a pasta aparecer.
- Editar e salvar a config vale em até 15 s, sem reiniciar o daemon.
- Custo: uma dependência externa, uns 290 KB a mais no binário e mais um item para
  acompanhar em segurança.
- Custo: com a config quebrada ao subir, o card fica vazio até você olhar o log ou
  rodar `config check`.
- Custo: trabalhar numa pasta ignorada pode deixar o card vazio por completo.

## Alternativas descartadas

- Parser próprio de um subconjunto do TOML: sem dependência, mas um TOML válido podia
  ser recusado ou, pior, lido errado numa config de privacidade.
- Reler só ao subir o daemon, ou por `SIGHUP`: exigiria reiniciar o daemon ou lembrar
  de mandar o sinal.
- `allow` acrescentando à lista padrão: não daria para remover um padrão sem uma
  segunda chave.
- Esconder só o nome da pasta ignorada: o card revelaria a atividade, contra o que o
  README promete.
- Usar os padrões quando a config é inválida: um `ignore_dirs` quebrado viraria
  vazamento silencioso.

## Quando revisar

- Se a `BurntSushi/toml` ficar 12 meses sem release ou tiver uma vulnerabilidade
  publicada (o mesmo gatilho do ADR-0005).
- Se você quiser acrescentar programas sem repetir a lista: aí entra uma chave
  `allow_extra` ou `deny`.
- Quando a allowlist padrão for revista (você comentou que quer decidir isso depois,
  inclusive o `docker`).
