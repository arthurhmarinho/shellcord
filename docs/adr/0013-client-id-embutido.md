# ADR-0013: Client ID da aplicação embutido no binário

- Status: Aceito
- Data: 2026-11-12

## Contexto

O handshake exige o Client ID de uma aplicação do Discord Developer Portal. No teste da
fase 1, ele foi passado por flag para não ficar fixo no código de um teste. Para o
produto, o binário precisa funcionar sem configuração.

## Decisão

O Client ID da aplicação "shellcord" (`1570271223571812512`) é uma constante em
`cmd/shellcord/main.go`. A variável de ambiente `SHELLCORD_CLIENT_ID` o sobrescreve.

## Consequências

- Funciona sem configuração.
- O Client ID não é segredo: ele identifica a aplicação, não o usuário, e o Discord o
  expõe em toda atividade (`application_id`).
- Custo: quem fizer fork e quiser outro nome no card ("Jogando X") precisa da variável
  ou de recompilar.
- Custo: as imagens do card ficam presas a esta aplicação no Developer Portal.

## Alternativas descartadas

- Client ID obrigatório na config: atrito na instalação sem ganho de segurança.

## Quando revisar

- Se a aplicação for recriada no Developer Portal (o ID muda).
- Se a config TOML (fase 5) ganhar um campo para isso: aí a ordem de precedência
  (variável, config, constante) precisa ser decidida.
