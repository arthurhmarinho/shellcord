# ADR-0011: Agregação de fontes de mesma prioridade

- Status: Aceito
- Data: 2026-11-12

## Contexto

Várias fontes podem estar no mesmo nível de prioridade (ADR-0004), como duas sessões
do Claude Code trabalhando ou comandos rodando em dois terminais. O Discord mostra uma
atividade só.

## Decisão

Agregar: o card mostra o nível mais alto, com a contagem de fontes nesse nível quando
houver mais de uma (ex.: "Claude Code · 2 sessões"). A pasta e o cronômetro vêm da
fonte mais antiga do nível.

## Consequências

- O card fica estável: o cronômetro não reinicia e o texto não troca a cada 15 s.
- Custo: só a pasta da fonte mais antiga aparece, e as outras ficam ocultas.
- Custo: o texto agregado depende de cada tipo de fonte. O formato exato de cada nível
  ainda será definido (fases 3 e 4).

## Alternativas descartadas

- A mais recente vence: simples, mas com duas fontes ativas o card alterna a cada 15 s
  e o cronômetro reinicia.
- A mais antiga vence sem contagem: estável, mas esconde que há mais atividade.

## Quando revisar

- Se no uso real você quiser ver a pasta da sessão ativa mais recente em vez da mais
  antiga.
- Se a contagem nunca passar de 1 em semanas de uso: aí a lógica de agregação pode ser
  simplificada.
