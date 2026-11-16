# Contrato de integração Jarvis ↔ OpenCode

Estado: investigação do SDK `@opencode/client` 2.2.7; **nenhuma operação de
escrita foi habilitada**. Este documento registra os limites que devem orientar
a próxima fatia C2.

## Operações confirmadas no cliente V2

- `session.create` aceita `location.directory` e `permissions`; o diretório deve
  vir exclusivamente do mapeamento de projeto allowlisted no bridge.
- `session.prompt` aceita `sessionID`, texto, anexos/menções, `delivery`, `resume`
  e um `id` opcional. A resposta é `SessionInboxUser`, não a resposta final do
  agente. Portanto, HTTP 2xx significa que a entrada foi aceita/enfileirada, não
  que o trabalho terminou.
- `message.list({sessionID, limit, order, cursor, type})` devolve mensagens
  persistidas e cursor. `session.get`, `session.inbox.list`, `session.diff` e
  `session.interrupt` suportam recuperação/estado; os eventos continuam sendo
  live-only e não substituem consultas após reconexão.
- `permission.list/get/reply` consultam pedidos pendentes por sessão. `reply`
  aceita `once`, `always` ou `reject`; `always` pode salvar autorização e não
  deve ser oferecido como aprovação normal na UI móvel.

## Contrato de segurança para escrita (a implementar)

1. A entrada do navegador usa somente IDs Jarvis allowlisted e texto simples
   limitado em tamanho. Não aceitar `directory`, URL de arquivo, anexos, agente,
   skill, delivery livre, regras de permissão ou ID OpenCode arbitrário do
   cliente. Anexos/skills/agentes ficam fora da primeira versão.
2. Para toda operação que recebe `sessionID`, o bridge busca a sessão e valida
   que sua localização/diretório canônico corresponde ao projeto Jarvis
   selecionado. Não confiar apenas no formato do ID ou em metadados enviados pelo
   browser.
3. Criar sessões com regras explícitas de menor privilégio e testar sua semântica
   contra OpenCode 2.2.7. Permissões do OpenCode são uma camada de autorização do
   executor, não sandbox do sistema operacional. Projetos devem continuar sem
   segredos privilegiados e qualquer acesso a rede/shell precisa de aprovação
   deliberada. Não alegar isolamento de filesystem sem sandbox externo.
4. Expor pedidos de permissão com ação, recursos, mensagem e projeto/sessão
   claramente identificados. Responder por padrão apenas `once` ou `reject`;
   rejeitar `always` no contrato público inicial. Revalidar sessão e pedido
   pendente imediatamente antes da resposta.
5. `session.prompt` é uma mutação com efeito externo. Registrar uma operação
   Jarvis antes de chamá-la e separar estados `submitting`, `accepted`, `running`,
   `completed`, `failed` e `uncertain`. Uma falha/timeout após iniciar a chamada
   não autoriza repetição automática. `id` opcional não será tratado como chave
   idempotente até isso ser confirmado em teste/documentação de servidor.
6. Em `uncertain`, consultar inbox, mensagens e estado persistido para reconciliar
   antes de habilitar retry manual. Se não for possível provar se o prompt foi
   aceito, manter estado incerto e pedir decisão explícita ao usuário.
7. Retornar apenas texto final do assistente, eventos de progresso permitidos e
   diff da sessão selecionada; não enviar reasoning, dados nativos do provedor,
   paths absolutos ou conteúdo de ferramenta sem uma decisão explícita de produto.

## Próximas verificações antes de habilitar escrita

- Validar no OpenCode 2.2.7 as regras `permissions` em sessão recém-criada e como
  wildcard `ask`/`deny` interage com configuração global e ferramentas.
- Provar que uma sessão existente pode ser atribuída ao projeto exclusivamente
  pela localização retornada por `session.get`.
- Verificar se o `id` de `session.prompt` é idempotente no servidor; até lá,
  implementar reconciliação e nunca retry automático.
- Testar eventos, consultas persistidas, paginação, respostas vazias e sessão em
  execução/interrompida. Não fazer prompt de teste real sem confirmação explícita.
- Adicionar persistência Jarvis para vínculo projeto↔sessão e operação de envio
  antes de expor endpoint mutável no Core/UI.
