# Contrato de integração Jarvis ↔ OpenCode

Estado: integração local read/write implementada atrás de opção explícita;
**não foi executada tarefa real no OpenCode neste ambiente**. Este documento
registra os limites que devem orientar a qualificação no computador-alvo.

## Fronteira de acesso escolhida

- Não haverá login/conta Jarvis no fluxo atual. A UI web será acessível na
  máquina local ou por Tailscale, com regras de rede limitadas aos dispositivos
  autorizados. Validar essas regras com um dispositivo permitido e outro negado;
  não presumir que todos os membros da tailnet devem acessar o serviço.
- O frontend é a única entrada remota. Core, bridge e serviço OpenCode ficam em
  loopback; não publicar suas portas nem adicionar CORS amplo. O bearer token
  entre Core e bridge é credencial interna e não substitui a fronteira da rede.
- O app Android futuro não terá login Jarvis. Antes de conceder acesso remoto,
  parear cada instalação por chave privada no Android Keystore e permitir
  revogação no servidor. O OAuth do Google Home, se usado, permanece uma
  autorização separada para a conta/ecossistema Google.
- Implementado: a tela/rotas de login foram removidas e script/Core já não exigem
  credenciais de usuário para habilitar o bridge. O Core não autentica usuários;
  mantenha-o isolado atrás do frontend e da rede privada.

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
- `session.form.list/get/reply` expõe formulários interativos do agente. O canal
  WhatsApp repassa texto, rótulos, descrições e escolhas; um número seleciona a
  opção correspondente e texto livre responde de forma personalizada. A UI web
  ainda não apresenta esses formulários como controles estruturados.
- `session.interrupt` solicita que o OpenCode pare a execução atual. O aceite da
  chamada não garante parada instantânea nem desfaz arquivos/efeitos já
  produzidos; após resposta ou erro incerto, consultar novamente a sessão. Nunca
  repetir automaticamente uma interrupção cujo resultado seja incerto.

## Contrato de segurança para escrita implementado

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
5. `session.prompt` e `session.interrupt` são mutações com efeito externo.
   Registrar cada operação
   Jarvis antes de chamá-la e separar estados `submitting`, `accepted`, `running`,
   `completed`, `failed` e `uncertain`. Uma falha/timeout após iniciar a chamada
   não autoriza repetição automática. Interromper não reverte alterações já
   realizadas. `id` opcional não será tratado como chave
   idempotente até isso ser confirmado em teste/documentação de servidor.
6. Em `uncertain`, consultar inbox, mensagens e estado persistido para reconciliar
   antes de habilitar retry manual. Se não for possível provar se o prompt foi
   aceito, manter estado incerto e pedir decisão explícita ao usuário.
7. Retornar apenas texto final do assistente, eventos de progresso permitidos e
   diff da sessão selecionada; não enviar reasoning, dados nativos do provedor,
   paths absolutos ou conteúdo de ferramenta sem uma decisão explícita de produto.

## Verificações pendentes antes de habilitar escrita

- Validar no OpenCode 2.2.7 as regras `permissions` em sessão recém-criada e como
  wildcard `ask`/`deny` interage com configuração global e ferramentas.
- Provar que uma sessão existente pode ser atribuída ao projeto exclusivamente
  pela localização retornada por `session.get`.
- Verificar se o `id` de `session.prompt` é idempotente no servidor; até lá,
  implementar reconciliação e nunca retry automático.
- Testar eventos, consultas persistidas, paginação, respostas vazias e sessão em
  execução/interrompida. Não fazer prompt de teste real sem confirmação explícita.
- A persistência do vínculo projeto↔sessão e do journal de operações está
  implementada em migration aditiva. Validar crash/restart no computador-alvo.
- O flag está desligado por padrão tanto no Core quanto no bridge. Não habilitar
  até concluir as verificações acima em uma instalação real e projeto descartável.

## Estado e sequência restante

1. Login Jarvis e dependência de inicialização do bridge removidos. Manter
   serviços internos em loopback; configurar Tailscale Serve e grants/ACLs por
   dispositivo, testando acesso permitido e negado.
2. Validar o fluxo `scripts/jarvis` → bridge → Core → `/code`, inclusive
   indisponibilidade e reconexão, na máquina-alvo.
3. Fechar os testes de contrato com OpenCode 2.2.7: permissões efetivas,
   correlação/id do prompt, paginação, respostas vazias e reconciliação após
   timeout/restart. Usar mocks onde possível. Qualquer tarefa real deve usar
   projeto descartável e autorização explícita do usuário.
4. Validar as regras Tailscale com um dispositivo autorizado e outro negado; o
   serviço de desenvolvimento não foi executado por esta implementação.
5. Interrupção explícita está implementada via UI/Core/bridge e journal; validar
   comportamento e reconciliação contra OpenCode real. Depois da qualificação,
   avaliar eventos ao vivo e resolução manual de operações incertas. Não repetir
   automaticamente nenhuma operação cujo resultado permaneça desconhecido.

Restrição de custo: orçamento incremental R$ 0. Não adicionar chave com cobrança
por uso nem fallback pago. Usar apenas uma conta/provedor que o usuário já tenha
e cujo uso esteja coberto pelo plano; não assumir cota ou disponibilidade.
