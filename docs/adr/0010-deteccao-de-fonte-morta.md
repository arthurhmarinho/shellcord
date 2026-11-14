# ADR-0010: Detecção de fonte morta por PID e starttime

- Status: Aceito
- Data: 2026-11-12

## Contexto

Uma fonte pode morrer sem avisar: terminal fechado à força, `SessionEnd` que não roda,
Claude Code morto. Sem detecção, o card fica preso em "trabalhando".

O que foi verificado em 2026-11-12, com o Claude Code 2.1.322: um processo disparado
pelo Claude Code tem como ancestral um processo com `comm` igual a `claude`, a um nível
(`zsh -c ...` é filho do `claude`).

Verificado em 2026-11-14 (fase 3), com uma sessão `claude -p --settings <arquivo>`:
os 6 hooks disparados (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop,
SessionEnd) rodam como `hook → sh → claude`, ou seja, o `claude` fica 2 níveis acima do
processo do hook.

## Decisão

- Cada fonte se identifica com `pid` e `starttime` (campo 22 de `/proc/<pid>/stat`).
  O starttime evita confundir a fonte com outro processo que reutilizou o mesmo PID.
- **fish:** o cliente manda o PID do fish (`$fish_pid`) e o starttime dele.
- **Claude Code:** o cliente sobe pela árvore lendo `/proc/<pid>/stat` até achar um
  ancestral com `comm` igual a `claude`, com profundidade máxima de 8, e manda o PID e o
  starttime desse ancestral junto com o `session_id`.
- **Fallback:** se a subida não achar o `claude`, o evento vai sem PID e a fonte expira
  por TTL (10 min sem eventos).
- O daemon confere se cada fonte está viva (`/proc/<pid>/stat` existe e o starttime
  bate) antes de calcular o estado a enviar, ou seja, no máximo uma vez a cada 15 s.

## Consequências

- Fontes mortas somem no máximo uma janela (15 s) depois de morrer.
- Custo de desempenho: no cliente, de 1 a 8 leituras de arquivos do `/proc` por evento
  (microssegundos). No daemon, uma leitura por fonte a cada 15 s.
- Custo: depende do nome do processo do Claude Code (`claude`) e de o hook ser
  descendente dele. Se isso mudar, o fallback por TTL segura, mas fica menos preciso.
- Custo: só funciona no Linux (`/proc`), o que é aceitável para este projeto.

## Alternativas descartadas

- TTL para o Claude Code: mais simples e independente da versão, mas uma sessão morta
  ficaria "trabalhando" por até 10 minutos.
- Só PID, sem starttime: um PID reutilizado faria uma fonte morta parecer viva.
- Heartbeat das fontes: exigiria um processo extra por fonte.

## Quando revisar

- Na fase 3: confirmar com um hook de teste que o `claude` aparece na cadeia de
  ancestrais dos hooks. Se não aparecer, este ADR é substituído.
- Se uma atualização do Claude Code mudar o `comm` do processo (o fallback por TTL
  passaria a ser usado sempre; um log de debug no daemon deve avisar).
