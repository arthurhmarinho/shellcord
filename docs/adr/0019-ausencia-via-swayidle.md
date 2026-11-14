# ADR-0019: Ausência do usuário via swayidle

- Status: Aceito
- Data: 2026-11-14

## Contexto

Um comando rodando não tem limite de tempo. Isso está certo para um `nvim` em que você
trabalha por horas, e errado para um `htop` esquecido. O daemon não consegue distinguir
os dois pelos eventos do shell: nenhum deles gera evento enquanto roda. O sinal que
distingue os dois é a presença de alguém no teclado.

## Decisão

- Novos subcomandos `shellcord away` e `shellcord back`, que mandam um evento da fonte
  `system`.
- O `swayidle` chama os dois: `swayidle -w timeout 600 'shellcord away' resume
  'shellcord back'`. Os 600 s batem com o limite de inatividade do daemon.
- Enquanto você está ausente, o card mostra **só "Claude Code trabalhando"**: o Claude
  está de fato trabalhando, e o resto seria um terminal esquecido aberto. Todo o resto
  some e volta como estava quando você retorna, com os cronômetros originais.

## Consequências

- O `htop` esquecido some depois de 10 minutos sem teclado nem mouse. O `nvim` em que
  você trabalha continua.
- Custo: depende do `swayidle` e do protocolo de inatividade do Wayland. Sem ele, o
  comportamento é o de antes (comando rodando sem limite).
- Custo: o estado de ausência vive só na memória. Se o daemon reiniciar enquanto você
  está ausente, o card volta a aparecer até o próximo `away`.
- Custo: um vídeo em tela cheia pode inibir o idle (`-w` com `inhibit`), e aí você conta
  como presente.

## Alternativas descartadas

- Sem limite: o `htop` esquecido fica no card para sempre.
- Limite fixo para comandos (ex.: 60 min): esconde o `nvim` em que você está
  trabalhando.
- Esconder também o Claude trabalhando: mais conservador, mas apaga uma atividade real.
  Se preferir assim, é uma linha em `state.Summary`.

## Quando revisar

- Se você trocar o sway por outro compositor: o `swayidle` sai e outro gatilho de
  inatividade precisa chamar `away`/`back`.
- Se você preferir que o Claude trabalhando também suma quando você sai.
