# Registros de decisão de arquitetura (ADRs)

Cada arquivo registra uma decisão: o contexto, o que foi decidido, quanto custa, o que
foi descartado e um gatilho concreto para revisar.

Para mudar uma decisão, crie um ADR novo que substitua o antigo e marque o antigo como
"Substituído por ADR-NNNN". Não reescreva um ADR aceito.

| #    | Decisão                                                         | Status  |
|------|-----------------------------------------------------------------|---------|
| 0001 | [Go e protocolo IPC do Discord escrito à mão](0001-go-e-ipc-a-mao.md) | Aceito |
| 0002 | [Daemon residente e cliente curto no mesmo binário](0002-daemon-e-cliente.md) | Aceito |
| 0003 | [Coalescência, reconexão e limpeza por inatividade](0003-coalescencia-reconexao-inatividade.md) | Aceito |
| 0004 | [Prioridade das fontes de presença](0004-prioridade-das-fontes.md) | Aceito |
| 0005 | [Privacidade por padrão e config em TOML](0005-privacidade-e-config-toml.md) | Aceito |
| 0006 | [Arranque do daemon: sway e autoarranque pelo cliente](0006-arranque-do-daemon.md) | Aceito |
| 0007 | [Descoberta do socket do Discord, inclusive Flatpak](0007-descoberta-do-socket.md) | Aceito |
| 0008 | [Testes exploratórios fora do repositório](0008-testes-fora-do-repositorio.md) | Aceito |
| 0009 | [Transporte de eventos por datagrama Unix](0009-transporte-de-eventos-datagrama.md) | Aceito |
| 0010 | [Detecção de fonte morta por PID e starttime](0010-deteccao-de-fonte-morta.md) | Aceito |
| 0011 | [Agregação de fontes de mesma prioridade](0011-agregacao-de-fontes.md) | Aceito |
