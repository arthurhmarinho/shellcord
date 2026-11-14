# ADR-0004: Prioridade das fontes de presença

- Status: Substituído por ADR-0016
- Data: 2026-11-11

## Contexto

O Discord mostra uma atividade por aplicação. Várias fontes podem estar ativas ao mesmo
tempo: sessões do Claude Code, comandos em vários terminais e shells parados.

## Decisão

Ordem de prioridade, da maior para a menor:

1. Claude Code trabalhando;
2. comando rodando no terminal;
3. shell parado;
4. nada (presença limpa).

## Consequências

- O card reflete a atividade mais "interessante" do momento.
- Custo: com várias sessões no mesmo nível, ainda falta uma regra de desempate (ex.: a
  mais recente). Ela será definida na fase 2 ou 3.
- Custo: o daemon precisa saber quando uma fonte morreu sem avisar (terminal fechado,
  sessão do Claude encerrada à força). Isso exige expiração por fonte.

## Alternativas descartadas

- Última atualização vence: um `ls` em outro terminal esconderia o Claude Code
  trabalhando.
- Combinar várias fontes em um texto só: os campos `details` e `state` são curtos e o
  resultado ficaria ilegível.

## Quando revisar

- Se surgir uma fonte nova (ex.: editor) que não se encaixe na ordem.
- Se no uso real o card mostrar com frequência algo que você não considera o principal.
