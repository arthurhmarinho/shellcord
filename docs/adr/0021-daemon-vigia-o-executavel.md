# ADR-0021: O daemon se reexecuta quando o binário é atualizado

- Status: Aceito
- Data: 2026-11-14

## Contexto

Um `go install` troca o arquivo do binário, mas o daemon continua rodando a imagem
antiga em memória (`/proc/<pid>/exe` aponta para `(deleted)`). Isso aconteceu duas
vezes na fase 4: o daemon velho recusou os eventos `away`/`back`, que ele não conhecia.

## Decisão

- Na checagem de 15 s, o daemon compara o próprio executável em disco (inode, data de
  modificação e tamanho) com o que estava lá quando ele subiu.
- Ele só age quando a identidade nova se repete em duas checagens seguidas, para não
  reexecutar um arquivo ainda sendo escrito.
- Aí o `daemon.Run` libera tudo (conexão com o Discord, socket e lock) e devolve
  `ErrRestart`. O `main` chama `syscall.Exec` com os mesmos argumentos: o PID continua
  o mesmo e a imagem passa a ser a nova.

## Consequências

- Atualizar é só rodar `go install` de novo. Verificado em 2026-11-14: o daemon se
  reexecutou 30 s depois da troca, com o mesmo PID.
- Cada daemon compara só o próprio arquivo, então dois binários diferentes em uso não
  derrubam um ao outro.
- Custo: o estado em memória se perde na troca. As fontes só voltam no próximo comando
  ou hook, e um `away` em curso é esquecido. A conexão com o Discord reabre, então o
  card pisca uma vez.
- Custo: até 30 s rodando a versão velha depois do `go install`.

## Alternativas descartadas

- O cliente manda a versão e o daemon sai se for diferente: detecta na hora, mas dois
  binários diferentes em uso se derrubariam em loop, e o protocolo mudaria.
- `shellcord restart` manual: simples, mas exige lembrar, e um `go install` solto
  continua deixando o daemon velho.
- Salvar o estado em `$XDG_RUNTIME_DIR/shellcord/state.json` antes da troca: evitaria a
  piscada, mas é mais código para um caso raro.

## Quando revisar

- Se a piscada do card ou a perda do `away` na atualização incomodarem: aí entra o
  `state.json`.
- Se o binário passar a ser instalado por um gerenciador de pacotes que troca o arquivo
  de outro jeito (ex.: edição no lugar, sem trocar o inode).
