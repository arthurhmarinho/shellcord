# ADR-0022: `shellcord install` e `uninstall` com diff e confirmação

- Status: Aceito
- Data: 2026-11-14

## Contexto

A integração mexe em três arquivos do usuário: `~/.claude/settings.json`,
`~/.config/fish/config.fish` e `~/.config/sway/config`. Editar à mão é onde se erra. Um
instalador que estraga o arquivo de outra pessoa é pior que não ter instalador.

## Decisão

- Subcomandos `install` e `uninstall`, com alvos opcionais (`claude`, `fish`, `sway`;
  sem alvo, os três), `-dry-run` e `-yes`.
- Para cada arquivo, o comando planeja o texto novo, mostra o diff e pergunta antes de
  gravar. O original vai para `<arquivo>.shellcord.bak`. A escrita é atômica (arquivo
  temporário + rename), mantém a permissão e segue symlinks, para dotfiles gerenciados
  continuarem symlink.
- **fish e sway:** um bloco entre `# >>> shellcord >>>` e `# <<< shellcord <<<`.
  Reinstalar troca o bloco no lugar. Uma integração feita à mão fora dos marcadores é
  reconhecida: o `install` não duplica e o `uninstall` não a remove (avisa).
- **Claude Code:** o `settings.json` é lido como um objeto que **preserva a ordem das
  chaves** e o texto dos valores que o shellcord não toca. Os hooks do shellcord são
  reconhecidos pelo comando (`… shellcord … hook claude`), e os hooks de outros
  programas ficam intactos. A saída usa recuo de 2 espaços.
- Todas as integrações usam o caminho absoluto do binário, tirado de `os.Executable`.

## Consequências

- Instalar ou remover é um comando, sempre com o diff à vista.
- Verificado em 2026-11-14 contra os arquivos reais: o `install` não mudou nenhum byte
  do `settings.json` já integrado e reconheceu a integração feita à mão no fish e no
  sway.
- Custo: umas 750 linhas de código e 300 de teste (o editor de JSON que preserva ordem e o
  diff por maior subsequência comum).
- Custo: um `settings.json` com recuo diferente de 2 espaços é reformatado. O diff
  mostra isso antes.
- Custo: caminho do binário com espaço ou aspas é recusado no sway.

## Alternativas descartadas

- `install.sh`: depende do `jq` para o JSON, é difícil de testar e teria que reproduzir
  os marcadores e o diff.
- Só instruções no README: nenhum código, mas cada pessoa edita três arquivos à mão.
- Decodificar o `settings.json` num `map`: simples, mas embaralharia a ordem das chaves
  do usuário.

## Quando revisar

- Se o Claude Code mudar o formato dos hooks no `settings.json`.
- Se entrar outro shell ou compositor: cada um vira um alvo novo.
