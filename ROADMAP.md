# ROADMAP.md

Roadmap vivo do projeto Jarvis.

Este arquivo deve ser atualizado quando funcionalidades forem concluidas, iniciadas ou replanejadas.

## Avaliação de prontidão — 2026-11-16

**Conclusão:** estamos perto em termos de implementação, mas ainda não temos
desenvolvimento remoto comprovado/operacional. A primeira fatia completa existe
no código e a escrita está protegida por opt-in; ainda não foi qualificada contra
uma instalação OpenCode real nem por um telefone na tailnet. Não anunciar como
recurso móvel pronto nem ativar `JARVIS_OPENCODE_WRITE_ENABLED` antes dos gates
abaixo.

### Evolução observada

- A primeira versão conectava bridge, Core e `/code` somente para leitura.
- A fronteira escolhida mudou de login Jarvis single-user para acesso local/
  tailnet sem conta Jarvis. O frontend valida `Origin` em mutações; Tailscale e
  suas regras por dispositivo passam a ser a fronteira de acesso remoto. Essa
  configuração ainda não foi exercitada.
- A versão atual acrescenta sessão Jarvis↔OpenCode persistida, operações com
  estados e ID de correlação, criação de sessão, prompt, consulta/reconciliação de
  resposta/diff e aprovações `once`/`reject`. Core e bridge exigem o flag de escrita;
  ambos ficam desligados por padrão. Testes automatizados cobrem contratos com
  mocks, não comportamento real do OpenCode.
- Continuação é por polling. A interrupção explícita agora existe na UI/Core/bridge
  com journal e reconciliação por estado da sessão; falta validá-la no OpenCode
  real. Stream de eventos permanece posterior.

### Gate para declarar “programar à distância” utilizável

1. Configurar Tailscale Serve e grants/ACLs; provar acesso de um dispositivo
   autorizado e bloqueio de outro, sem expor Core/bridge/OpenCode.
2. No computador-alvo, executar `scripts/jarvis` com OpenCode 2.2.7 e verificar
   `/code` em modo read-only e a allowlist.
3. Em projeto descartável e com autorização explícita, habilitar escrita e
   completar pelo telefone: criar sessão, enviar tarefa, aprovar uma ação uma vez
   ou rejeitá-la, receber resposta/diff e continuar a sessão.
4. Simular perda de conexão/reinício e confirmar que o journal reconcilia estados
   sem repetir prompt/efeito automaticamente; validar crash/restart do Core.

**Troca de foco:** depois desse gate, é razoável tirar OpenCode da prioridade
diária e retomar a trilha IoT quando houver Android. Hoje IoT continua bloqueada
para integração física: existe somente o probe Android read-only compilado; OAuth,
descoberta real e validação da lâmpada não foram executados. Sem Android, pode-se
retomar apenas planejamento/documentação, não afirmar progresso de integração.
Detalhes e sequência em A0–A5 abaixo.

## Prioridade atual — integração Jarvis ↔ OpenCode

Objetivo: permitir que o usuário programe pelo celular usando a interface do
Jarvis. O Jarvis recebe tarefas, encaminha-as ao OpenCode executando no
computador onde estão os projetos, apresenta progresso/resposta/alterações e
permite continuar sessões e responder a pedidos de aprovação.

Decisão de direção: **integrar o OpenCode V2 como executor de tarefas de código;
não construir um harness completo próprio nesta etapa**. O Jarvis continua sendo
a interface, o orquestrador e o registro da delegação. Isso não
pressupõe que OpenCode controle genericamente janelas ou qualquer aplicativo do
desktop.

### Estado conhecido

- Jarvis tem Core Go, frontend Next.js, agente de voz Python, conversas/eventos e
  journal no PostgreSQL. `/api/v2` atualmente autoriza somente `time` e
  `calculator`; ferramentas de sistema ficam bloqueadas por padrão.
- O Core não autentica usuário final. **O login Jarvis não faz parte do objetivo
  atual e não deve ser requisito para usar o app.** O acesso web deve
  ficar limitado à máquina local ou à tailnet, com Core/backend em loopback e
  acessível pelo navegador via frontend. O controle de acesso por dispositivo na
  tailnet ainda precisa ser configurado e validado.
- OpenCode V2 fornece API de sessões, mensagens, eventos, diff e permissões, além
  de `@opencode/client` TypeScript e serviço local. Eventos do cliente são ao vivo,
  sem replay/reconexão automática; após desconexão, o Jarvis deverá consultar o
  estado persistido da sessão.
- Integração implementada em `opencode-bridge`: Node/TypeScript com
  `@opencode/client` 2.2.7, bind loopback, bearer token e allowlist explícita.
  Rotas read e write estão ligadas ao Core/UI; escrita exige
  `JARVIS_OPENCODE_WRITE_ENABLED=true` e fica false por padrão. Os contratos
  foram testados com mocks; validação com OpenCode real e acesso pelo telefone
  ainda não foram feitos.

### Fase C0 — acesso privado sem login Jarvis

Estado: o login Jarvis foi removido e a rede privada é a fronteira de acesso
escolhida; configuração e validação por dispositivo ainda pendentes.

- Confirmar OpenCode V2 e requisitos de instalação/serviço no computador-alvo.
- Restringir a interface web à máquina local e à tailnet. No acesso remoto, usar
  Tailscale Serve/reverse proxy para encaminhar ao frontend local; não publicar
  frontend, Core ou bridge na internet.
- Configurar grants/ACLs para os dispositivos escolhidos e confirmar na prática
  que um dispositivo autorizado conecta e outro dispositivo da tailnet não
  conecta. Não assumir que pertencer à mesma tailnet, por si só, seleciona um
  dispositivo específico.
- Implementado: o app não tem tela/rotas de login e o script/Core habilitam o
  bridge sem credenciais de usuário. O Core não autentica usuário; mantenha Core
  e bridge em loopback e não os exponha diretamente ao navegador/dispositivo.
- Para o futuro app Android/probe, não criar conta Jarvis nem tela de login.
  Antes de permitir chamadas remotas ao Core, definir pareamento por instalação
  com chave mantida no Android Keystore, revogação e autorização no servidor.
  OAuth/consentimento Google Home, se necessário para as Home APIs, é separado e
  não equivale a uma conta Jarvis.
- Implementado no bridge: lista explícita `JARVIS_OPENCODE_PROJECTS`, caminhos
  canônicos e IDs resolvidos no servidor; nenhum path é aceito do navegador.
- Implementado para o fluxo host do script: bridge OpenCode e Core ficam em
  loopback e usam bearer token interno. A implantação pelo Compose não habilita
  o bridge; a topologia de proxy privado ainda não foi testada.

Critério de saída: somente dispositivos explicitamente permitidos alcançam a
interface Jarvis pela rede privada; o app não pede login Jarvis; Core e OpenCode
permanecem fora do acesso direto do cliente.

### Fase C1 — contrato e adaptador OpenCode

Estado: adaptador read/write integrado; build e testes com mocks passaram. Falta
validar o fluxo script→Core→frontend, o contrato contra OpenCode 2.2.7 real e o
acesso remoto por dispositivo.
Mantenha `JARVIS_OPENCODE_WRITE_ENABLED=false` até C0 e as verificações reais de
permissão, ID de prompt e reconciliação estarem concluídas.

- Implementado: adaptador TypeScript estreito usando `@opencode/client` e serviço
  OpenCode local. O Core Go não executa shell nem recebe credenciais dos
  provedores de IA.
- Implementado: listar projetos/sessões, criar sessão, enviar prompt, ler estado/
  mensagens/diff e responder pedidos de permissão com `once`/`reject`. Interrupção
  ainda pendente.
- Não aceitar caminho livre. O servidor resolve identificadores de projeto para
  diretórios previamente cadastrados.
- Estado da implantação: processo bridge separado, iniciado opcionalmente pelo
  `scripts/jarvis` no host. Compose não o inclui; deployment final e proxy móvel
  continuam sem validação. SDK embutido não foi escolhido.
- Versionar o contrato Jarvis↔adaptador e tratar indisponibilidade, timeout,
  respostas inválidas e versões incompatíveis.
- Adaptador valida OpenCode `2.x` via `Service.discover`, faz bind em
  `127.0.0.1`, exige token interno de pelo menos 32 bytes e aceita somente
  projetos configurados em `JARVIS_OPENCODE_PROJECTS`. Core valida URL loopback e
  token interno; isso não autentica requests diretamente no Core. O Core deve
  permanecer fora do alcance remoto. Compose não ativa o bridge; o fluxo atual é
  `scripts/jarvis` no host.
- Bridge esconde reasoning, payloads de ferramenta e paths absolutos; limita texto
  e diff. Toda sessão recebida é revalidada contra o diretório canônico do projeto.
- Escrita fica desligada por padrão. Sessões novas pedem `ask` para ferramentas;
  a UI oferece somente `once` e `reject`, nunca `always`.
- Investigação do contrato V2 registrada em `docs/opencode-integration.md`:
  prompt retorna item de inbox (não conclusão); mensagens têm cursor; permissões
  suportam `once`/`always`/`reject`. Semântica de permissões efetivas e do campo
  `id` ainda precisa de validação no servidor local.

Critério de saída: testes de contrato comprovam isolamento entre projeto/sessão,
validação de entrada, falhas e cancelamento, sem executar comandos reais.

### Fase C2 — delegação e retorno no Jarvis

Estado: primeira versão implementada e desligada por padrão; qualificação real
depende de C0 e dos testes de contrato C1 no computador-alvo.

- Implementado: selecionar projeto, criar sessão OpenCode e enviar texto pela UI.
- Implementado: migrations aditivas guardam vínculo projeto/sessão e journal de
  operações `submitting`, `accepted`, `running`, `completed`, `failed` e
  `uncertain`; a tentativa é persistida antes da chamada externa.
- Implementado: mostrar respostas textuais e diffs limitados, continuar sessão,
  acompanhar por consulta e reconciliar após reconexão. Não copia todo o histórico
  do OpenCode para o PostgreSQL.
- Implementado: timeout/crash não repete prompt nem sessão automaticamente. O
  usuário pode consultar/reconciliar; IDs repetidos com payload diferente geram
  conflito.
- Validar em OpenCode real o retorno de `session.prompt`, `message.list` e os
  limites de reconciliação antes de habilitar o flag de escrita.

Critério de saída: tarefa de teste iniciada no Jarvis aparece como sessão
persistente, retorna resposta/diff e aceita continuação sem abrir outra interface.

### Fase C3 — eventos, reconexão e aprovações

Estado: aprovações únicas/rejeição e polling/reconciliação básicos implementados;
stream de eventos e interrupção ainda pendentes.

- Usar eventos ao vivo para atualização de baixa latência, mas consultar mensagens,
  estado e diff da API após reconexão; não tratar stream como log durável.
- Implementar parada/interrupção explícita e comunicar que cancelar não desfaz
  efeitos já concluídos.
- Implementado: mostrar pedidos de permissão e oferecer somente “permitir uma
  vez” ou “rejeitar”; `always` não existe na UI/API Jarvis.
- Implementado: armazenar decisão e operação antes da resposta ao OpenCode. A
  sessão nova solicita aprovação (`ask`) para ferramentas.
- Pendência: validar a política efetiva contra configuração global do OpenCode,
  implementar interrupção explícita e adicionar eventos ao vivo.
- A execução do shell continua tendo autoridade do usuário local;
  permissões do agente não são sandbox do sistema operacional.

Critério de saída: aprovações aguardam decisão, rejeições persistem, reconexão
recupera a sessão e nenhuma tarefa incerta é repetida às cegas.

### Fase C4 — experiência móvel e validação ponta a ponta

Estado: planejada; depende de C0–C3.

- Otimizar para celular seletor de projeto/sessão, envio de prompt, histórico,
  status, retorno/diff e caixa de aprovações.
- Testar no desktop e telefone pela rede privada; incluir perda de rede, reinício
  da UI, adaptador indisponível, permissão pendente e interrupção.
- Verificar que portas e segredos do executor não ficam acessíveis fora da fronteira
  prevista e que as proteções existentes de `/api/v2` não foram ampliadas por acaso.

Critério de saída: usuário consegue iniciar e continuar tarefa de código pelo
celular, acompanhar o resultado e autorizar/negar ações sem contornar o Jarvis.

### Ordem recomendada para concluir OpenCode

1. **C0, acesso e configuração:** login Jarvis e sua dependência do bridge foram
   removidos. Configurar Tailscale Serve e regras por dispositivo, mantendo
   frontend como única entrada remota e Core/bridge/OpenCode em loopback.
2. **C1, leitura ponta a ponta:** verificar serviço OpenCode, script, Core e
   `/code` no computador-alvo; conferir allowlist, reconexão e falhas de serviço.
   Fazer também a prova de acesso pela tailnet e de bloqueio por um dispositivo
   não autorizado.
3. **C1, contrato de escrita:** concluir as verificações de propriedade de
   sessão, permissões e prompt incerto descritas em
   `docs/opencode-integration.md`. Usar serviço simulado para testes automáticos;
   sem prompt real até preparar um projeto descartável e autorizar a prova.
4. **C2, qualificação da escrita:** habilitar o flag só na máquina-alvo após C0 e
   testar sessão, aprovação única/rejeição, resposta/diff e recuperação em projeto
   descartável; não usar projeto com segredos para essa prova.
5. **C3, robustez:** implementar interrupção, stream de eventos e reconciliar
   permissões incertas; preservar a regra de nunca repetir tarefa sem confirmação.
6. **C4, celular:** validar telefone em Wi-Fi e fora de casa via Tailscale, perda
   de rede, reinício do app e revogação/bloqueio de dispositivo.

Restrição de custo: orçamento incremental **R$ 0**. Não configurar chave de API
com cobrança, fallback pago, serviço hospedado pago ou compra de hardware. Para
inferência, usar somente uma conta/provedor que o usuário já tenha e cujo uso
esteja coberto pelo plano; limites e disponibilidade devem ser tratados como
variáveis e não como garantia.

### Não objetivos desta entrega

- Construir um harness/loop de agente, executor de shell, snapshots e sistema de
  permissões completos do zero.
- Criar contas, login ou senha próprios do Jarvis para o usuário nesta etapa.
- Controlar genericamente mouse, teclado, janelas ou qualquer aplicativo gráfico.
- Expor OpenCode diretamente à internet, automatizar aprovações ou permitir
  diretórios arbitrários.
- Retomar nesta trilha a integração Google Home; ela permanece pausada até o
  usuário ter o Android disponível.

### Referências V2

- [SDK](https://opencode.ai/docs/build/sdk/), [client](https://opencode.ai/docs/build/client/),
  [API](https://opencode.ai/docs/api/), [Web/serviço](https://opencode.ai/docs/cli/web/),
  [permissões](https://opencode.ai/docs/permissions/) e
  [políticas](https://opencode.ai/docs/policies/).

---

## Trilha residencial pausada — 2026-11-15

Esta trilha está pausada enquanto o usuário não dispõe do Android. A0–A5 abaixo
preservam o checkpoint e os critérios prévios; não são a prioridade ativa e não
devem avançar por OAuth, discovery ou atuação física até retomada explícita.

## Prioridade anterior — avaliação residencial (histórico)

Esta seção substitui a ordem das antigas próximas etapas abaixo. As listas das
fases iniciais são histórico de entregas, não uma descrição completa do produto
atual. Arquitetura e limites da nova integração: `ARCHITECTURE.md`; decisões:
`DECISIONS.md`.

### Base disponível e evidências

- Implementado: laboratório `/lab`, conversa `/api/v2`, histórico/turnos/eventos
  no PostgreSQL, journal de ferramentas, cancelamento e pipeline de voz separado.
- Validado em 2026-11-14: GPT-6-Luna via Codex App Server 0.169.0 com OAuth em
  perfil isolado; resposta real e execução real de `calculator`. Amostras de
  2,72 s e 5,76 s, respectivamente, não constituem benchmark nem garantia de cota.
- A conversa atual autoriza apenas `time` e `calculator`; `system` fica bloqueada
  por padrão. Existe probe Android de diagnóstico em `android/home-probe`, mas não
  existe ponte residencial com o Core.
- Qualidade PT-BR, áudio em dispositivo físico, consumo/latência e mídia pela
  tailnet ainda precisam de evidências específicas. Não declarar voz de produção
  validada apenas pela existência do pipeline; ver `docs/voice-lab/README.md`.
- Memória de usuário e agentes continuam fora desta primeira entrega residencial.

### A0 — Viabilidade e preparação Android (checkpoint; pausada)

Estado: preparação de build concluída; OAuth e validação física aguardam o usuário
ter Android disponível. Fase pausada.

- Verificar requisitos vigentes das Home APIs: SDK/distribuição/licença, Android
  mínimo, Google Play Services, conta/casa elegíveis e eventuais hubs necessários.
  Resultado documentado em `docs/home-apis/README.md`.
- Google lista oficialmente Android Studio 2024.2.1+ e ADB como requisitos. O
  mantenedor obteve os artefatos Home beta e o probe compilou via Gradle Wrapper/JDK 17;
  isso não substitui o fluxo oficial nem valida execução no aparelho.
- Definir identidade do app, assinatura de desenvolvimento e registro OAuth/
  projeto Google, consentimento e usuários de teste conforme documentação atual.
  Configurações na conta Google e autorizações exigem participação do usuário.
- Usar app de exemplo oficial como referência para uma prova mínima; ainda não
  construir cliente completo de voz nem ponte permanente.
- Instalar APK em Android externo quando necessário, via ADB autorizado ou
  instalação manual. Não habilitar depuração nem alterar o aparelho sem anuência.
- Manter orçamento adicional R$ 0; não habilitar cobrança/fallback pago.

Progresso: SDK beta 1.11.1 instalado em `~/.m2/repository`; JDK 17 instalado pelo
mantenedor; sample oficial e app mínimo `android/home-probe` compilados com Gradle
Wrapper. O probe não oferece comandos: só descobre tipos de luz e exibe nomes/
cômodos. APK debug em `android/home-probe/app/build/outputs/apk/debug/app-debug.apk`
(~33 MB, não versionado). OAuth ainda não foi configurado; `adb devices` está
vazio e nenhum dispositivo doméstico foi consultado. Procedimento em
`docs/home-apis/README.md`.

Checkpoint concluído: probe compila com package/signature Jarvis próprios. OAuth,
consentimento e abertura em Android 10+ continuam pendentes até retomada.

### A1 — Descoberta, identificação e autorização da lâmpada

Estado: planejado; depende de A0.

- Consultar casas, cômodos, dispositivos e capacidades disponíveis nas Home APIs,
  inicialmente somente leitura e sem enviar comandos.
- Localizar a candidata “Luz Do Quarto”, vinculada via NovaDigital Smart, nome de
  origem “luzinha”. O usuário confirma o dispositivo, especialmente se houver
  nomes iguais. Não pedir caixa, MAC, ID virtual ou credenciais pelo chat.
- Associar apelido do Jarvis a identificador opaco da API, casa e capacidades
  observadas; nome de exibição não é identidade nem autorização.
- Validar comandos explícitos de ligar/desligar expostos pela API. A integração
  da conta NovaDigital com Tuya/Home Assistant não está comprovada.
- Se a lâmpada não aparecer ou não expuser os comandos necessários, parar esta
  rota e avaliar alternativas oficiais/localmente compatíveis com o usuário;
  não reconfigurar, restaurar ou desvincular a lâmpada automaticamente.

Critério de saída: dispositivo confirmado e capacidades registradas, sem atuação.
Essa é a porta de decisão antes de investir na ponte completa.

### A2 — Contrato e segurança da ponte Core ↔ Android

Estado: planejado; depende da viabilidade em A1.

- Definir protocolo versionado de descoberta/estado/comando/resultado, pareamento
  autenticado, revogação e canal privado protegido. Não expor portas públicas.
- Core mantém catálogo autorizado, políticas e journal; Android guarda a sessão
  Google e executa apenas comandos estruturados autorizados. Sem shell ou URL
  arbitrária gerados pelo modelo. Tokens/segredos não entram em logs ou Git.
- Selecionar transporte após testar rede local/tailnet e ciclo de vida Android.
  Definir conexão iniciada pelo app, heartbeat, expiração e reconexão; protocolo
  concreto e mecanismo de segundo plano ainda são decisões pendentes.
- Incluir ID de comando, validade, deduplicação, erros, revogação, estados
  enviados/confirmados/incertos e recuperação sem reexecutar ações antigas.
- Verificar tela bloqueada, segundo plano, suspensão, bateria e perda de rede.
  A disponibilidade da ponte depende do Android; não prometer controle permanente.
- Prever migrações aditivas para cadastro/journal se necessárias, preservando dados.

Critério de saída: contratos documentados e testes simulados aprovados para
autorização, revogação, duplicação, expiração, desconexão e resultado incerto.

### A3 — Primeira atuação controlada

Estado: planejado; depende de A2.

- Implementar adaptador Home APIs e ferramenta no registry, com habilitação
  explícita e allowlist de um dispositivo e duas ações: ligar/desligar.
- Aplicar a política tanto no fluxo conversacional quanto nos endpoints manuais;
  registrar a tool não deve contornar a autorização atual de `/api/v2`.
- Testar primeiro com simulador; depois executar teste físico autorizado pelo
  usuário e consultar estado. Evitar `toggle` e repetição automática após timeout.
- Diferenciar comando aceito de estado observado e relatar falhas com honestidade.

Critério de saída: ligar/desligar a lâmpada autorizada com resultados registrados;
pedidos para outros dispositivos são recusados e reconexão não repete comandos.

### A4 — Texto, voz e disponibilidade real

Estado: planejado; depende de A3.

- Integrar ferramenta ao motor e contexto de conversa, sem ampliar permissões do
  modelo. Pedido ambíguo exige esclarecimento, não escolha automática de cômodo.
- Validar primeiro texto e depois voz pelo pipeline existente; o MVP Android
  não precisa replicar a interface de voz.
- Testar interrupção, app fechado, autorização revogada, celular indisponível,
  estado desatualizado e resposta perdida após atuação. Cancelar a fala não desfaz
  uma ação já executada. Não prometer execução exatamente uma vez no dispositivo.

Critério de saída: pedido falado resulta em ação autorizada e resposta coerente
com o estado confirmado ou com a incerteza observada, sem regressões no laboratório.

### A5 — Expansão condicionada

- Somente após a lâmpada, inventariar capacidades reais da TV/Google TV/TV Box.
- Presença no Google Home não garante controle remoto completo, abertura de
  qualquer app ou simulação de botões. O tipo API `GoogleTVDevice` não comprova
  suporte a um aparelho Google TV específico.
- Automação de interface de apps e investigação de protocolos são escopos futuros
  separados; não são requisito nem promessa desta integração.

### Evidências e continuidade

Cada fase deve registrar versões, testes executados, resultado e bloqueios, sem
identificadores domésticos sensíveis. Testes físicos exigem dispositivo externo;
conta Google, consentimento e autorização de atuação ficam com o usuário.
Planejamento consolidado não equivale a SDK instalado, conta configurada ou
compatibilidade confirmada. Não iniciar A2/A3 se A1 falhar.

## Concluido

- Estrutura inicial do monorepo criada.
- Backend Go com Gin criado.
- Endpoint `GET /health` implementado.
- Health check verifica conexao com PostgreSQL.
- Configuracao do backend via environment variables.
- Encapsulamento inicial de PostgreSQL em `internal/database`.
- Dockerfile do backend com build multi-stage.
- Frontend Next.js com TypeScript e Tailwind criado.
- Tela inicial do frontend exibindo status do backend.
- Service frontend para consultar `/health`.
- Dockerfile do frontend com build multi-stage.
- Docker Compose com `postgres`, `backend` e `frontend`.
- PostgreSQL configurado com volume persistente.
- Migration inicial `001_initial_schema.sql` com tabela `app_metadata`.
- README com instrucoes de execucao.
- Separacao entre URL interna e URL publica da API no frontend.
- Estrutura reservada para `ai`, `voice`, `memory`, `tools`, `agents`, `services` e `repositories`.
- Contrato `LLM` criado em `backend/internal/ai`.
- `OllamaProvider` implementado atras da interface de IA.
- Configuracao de `OLLAMA_URL` e `OLLAMA_MODEL` por environment variables.
- Camada inicial de `services` criada para conversa por texto.
- Endpoint `POST /api/chat` implementado.
- Endpoint `POST /api/chat/stream` implementado com SSE.
- Tela simples de chat no frontend consumindo o endpoint do backend.
- Frontend renderiza resposta do chat em tempo real via stream.
- CORS local configuravel por `CORS_ALLOWED_ORIGIN`.
- Contrato `Tool` criado em `backend/internal/tools`.
- `ToolRegistry` criado para registrar, listar, buscar e executar ferramentas.
- `TimeTool` implementada para retornar data e hora atual.
- `CalculatorTool` implementada para calculos basicos.
- `ToolService` criado para orquestrar execucao de ferramentas via registry.
- Endpoints `GET /api/tools` e `POST /api/tools/execute` implementados.
- Tela simples de teste de ferramentas criada no frontend.
- `SystemTool` implementada com acoes pre-cadastradas e sem shell livre.
- Chat integrado ao `ToolRegistry` por roteamento local de tools obvias.
- `ChatService` decide entre streaming direto de resposta normal e execucao de tool com resposta deterministica.
- TimeTool, CalculatorTool e SystemTool podem ser acionadas automaticamente pelo chat.
- Streaming de texto preservado para mensagens que nao usam tools.
- Modulo `voice` criado com interfaces `SpeechRecognizer` e `VoiceProvider`.
- Provedores locais de voz por comando configuravel adicionados para STT e TTS.
- Endpoint `POST /api/voice/conversation` implementado para fluxo audio -> texto -> chat/tools -> voz.
- Interface principal redesenhada para experiencia prioritariamente por voz.
- Modo texto mantido recolhido como fallback.
- Interface principal refinada como HUD futurista com nucleo central animado.
- Chat e tools movidos para paineis laterais secundarios.
- Servidor STT local persistente adicionado para manter modelo carregado entre falas.
- VAD local no frontend adicionado para detectar fim de fala sem clique.
- Wake word simples por transcricao adicionada para ignorar audio sem chamada ao Jarvis.
- Modulo `voice-agent` criado como fronteira separada para evoluir voz realtime com Pipecat/LiveKit.
- `voice-agent` ja possui cliente do Jarvis Core via `POST /api/chat/stream`, modo console, servidor local de health/smoke test e testes do parser SSE.
- `voice-agent` emite tokens de sessao LiveKit para clientes desktop/Android via `POST /livekit/session`.
- Frontend chama o `voice-agent` por `NEXT_PUBLIC_VOICE_AGENT_URL`, entra na sala LiveKit e publica o microfone a partir do HUD principal.
- `voice-agent` possui participante realtime que entra na sala LiveKit, assina faixas de audio e contabiliza frames recebidos.
- `voice-agent` segmenta audio realtime por VAD simples, envia turnos para STT local e chama o Jarvis Core quando a wake word e detectada.

## Em Andamento

- Fundacao arquitetural do projeto.
- Documentacao viva para continuidade entre sessoes de IA e colaboradores.
- Definicao de fronteiras entre modulos futuros.
- Preparacao para futuras tools como web search, memoria, arquivos e automacao residencial.
- Preparacao para keyword spotting acustico e streaming de audio futuros.
- Migracao da experiencia principal de voz do fluxo batch para um agente realtime.

## Parcialmente Implementado

- `memory`: diretorio existe, mas nao ha modelo, repositorio ou API de memoria.
- `tools`: possui contrato, registry, TimeTool, CalculatorTool e SystemTool; ainda nao ha ferramentas externas avancadas.
- `agents`: diretorio existe, mas nao ha agentes autonomos.
- `ai`: possui interface `LLM` e provider inicial Ollama; ainda nao ha OpenAI ou Gemini.
- `voice`: possui contratos, provedores locais por comando/HTTP, servidor STT local, endpoint de conversa, VAD no frontend e UI de captura; depende de binarios/modelos locais de STT/TTS instalados no ambiente e fica como fallback batch.
- `voice-agent`: possui base separada para voz realtime, emissao de token LiveKit, integracao inicial com o HUD e participante que assina audio; ja transcreve turnos via STT HTTP local, mas ainda nao sintetiza audio de resposta de volta para a sala.
- `services`: possui chat com roteamento local para tools obvias e service de tools; ainda nao ha memoria ou agentes.
- `repositories`: diretorio existe, mas ainda nao ha repositorios de dominio.
- `hooks`: diretorio frontend existe, mas ainda nao ha hooks.

## Proximas Etapas da fase inicial (histórico; ordem substituída por A0–A5)

1. Definir modelo de dominio minimo para conversas por texto.
2. Criar migracoes versionadas para entidades reais do Jarvis.
3. Criar camada de repositories para acesso a dados.
4. Persistir historico de mensagens.
5. Criar UI de chat conversacional com historico local ou persistido.
6. Definir contrato de memoria persistente.
7. Adicionar mais testes automatizados para backend e frontend.
8. Definir padrao de observabilidade minimo, como logs estruturados e erros padronizados.
9. Adicionar configuracao seletiva de provedores de IA quando OpenAI ou Gemini forem implementados.
10. Evoluir tool calling por LLM para casos nao obvios sem prejudicar o streaming de mensagens normais.
11. Validar manualmente transporte realtime LiveKit + STT + Jarvis Core em ambiente local com rede Docker funcional.
12. Conectar TTS/audio de resposta do `voice-agent` de volta para a sala LiveKit e medir latencia por etapa.
13. Configurar e documentar modelos locais recomendados para Faster Whisper, Sherpa-ONNX, Piper/Kokoro e alternativas.
14. Evoluir wake word por transcricao para keyword spotting acustico local.
15. Criar cliente Android/APK apontando para o servico local via transporte realtime.

## Futuras Evolucoes

- Keyword spotting acustico para wake word.
- Sintese de voz.
- Transcricao de audio.
- Sistema de ferramentas executaveis com mais integracoes externas.
- Ferramentas futuras como WebSearchTool, HomeAssistantTool, MemoryTool e FileTool.
- Agentes autonomos com limites de seguranca.
- Suporte a OpenAI.
- Suporte a Gemini.
- Acesso remoto pela tailnet, limitado aos dispositivos escolhidos; sem login
  Jarvis no fluxo atual.
- Interface responsiva otimizada para celular.
- Painel de configuracoes do assistente.
- Historico de conversas e busca em memoria.

## Dependencias Entre Modulos

- `ai` ja expoe o contrato inicial `LLM`; novos provedores devem implementar essa interface ou uma evolucao compativel dela.
- `memory` depende de schema de banco, repositories e definicao de dados persistidos.
- `tools` expoe um `ToolRegistry` que deve ser usado por services, IA e agentes para executar ferramentas sem depender de implementacoes concretas.
- `agents` depende de `ai`, `tools`, `memory` e regras de permissao.
- `voice` batch depende de binarios/modelos locais configurados por env vars e usa o fluxo de chat existente.
- `voice-agent` depende do Jarvis Core via `POST /api/chat/stream`; transporte realtime deve permanecer desacoplado de tools e comandos diretos.
- UI de chat final depende de historico e modelo de conversa; a tela atual valida a integracao inicial com streaming.
- Acesso remoto depende de regras de rede privada e autorização dos dispositivos;
  não de conta ou senha Jarvis.

## Modulos Desenvolviveis em Paralelo

- Frontend de chat pode evoluir em paralelo usando o contrato `POST /api/chat`.
- Repositories e migrations podem evoluir em paralelo com UI, desde que contratos sejam documentados.
- Novos provedores de IA podem evoluir em paralelo atras da interface `LLM`.
- Documentacao e ADRs podem evoluir em paralelo com qualquer modulo.
- Observabilidade basica pode evoluir em paralelo com features de produto.
- Testes de handlers/services podem evoluir em paralelo com a UI.
- Voice batch pode evoluir em paralelo com UI e providers, desde que preserve os contratos `SpeechRecognizer` e `VoiceProvider`.
- Voice realtime pode evoluir em paralelo no `voice-agent`, desde que o backend Go continue sendo a autoridade para tools, memoria e permissoes.
- Agents devem aguardar definicao de tools, memory e limites de seguranca.

## WhatsApp — primeira etapa de texto

- Implementado em `backend/whatsapp` via `whatsmeow`, usando apenas o chat consigo mesmo da conta que corresponde ao número configurado. Texto comum aciona a resposta e a marca `Jarvis:` evita loops.
- Cada mensagem é processada isoladamente pelo GPT no perfil autenticado do Codex. A deduplicação é limitada à execução atual. O SQLite guarda o estado do dispositivo vinculado por número, permitindo parear outra conta com a mesma regra de chat consigo mesmo.
- Próximo teste: parear a conta escolhida e confirmar recebimento e envio na instalação real. A integração não oficial está sujeita a desconexões e mudanças do WhatsApp.
- Segunda etapa: definir vínculo com o histórico e a memória do Jarvis de maneira independente do número usado. Mensagens de voz e avaliação da Cloud API oficial ficam para depois.

## Atualização 2026-11-13 — Laboratório de voz contínua

Implementado:

- `/lab` com seleção de STT/TTS/voz/turn detection/provedor, histórico e exportação de eventos.
- Conversas, turnos, eventos SSE retomáveis e journal de ferramentas no PostgreSQL.
- Migrações aditivas, recuperação de turnos interrompidos e trava de instância do Core.
- SmallWebRTC, Silero, Smart Turn v3.2, áudio progressivo e descarte por geração.
- Adaptadores CPU Whisper base/small, Kokoro PT-BR e Pocket português.
- Provedores scripted, Groq, Ollama e Codex App Server experimental.
- Scripts de instalação/execução, corpus, WER, TTS isolado, recursos e smoke tests.

Pendências registradas em 2026-11-13 (reconciliadas pela prioridade atual acima):

- Downloads e inferência dos pesos STT/TTS bloqueados pela rede do executor.
- Licença individual da voz Pocket, enquadramento OAuth e elegibilidade/cota da conta.
- Inferência real de Groq/Ollama e comparação dos modelos reais elegíveis;
  ChatGPT/GPT-6-Luna teve resposta e calculadora reais validadas em 2026-11-14.
- Áudio no browser físico; Chromium gerenciado bloqueia UDP WebRTC neste executor.
- Notebook de 8 GB, qualidade PT-BR, metas de latência e interrupção acústica.
- ICE/Tailscale/Android em Wi-Fi e rede móvel; seleção final dos perfis.

Candidatos de segunda rodada permanecem condicionais, conforme a matriz em
`docs/voice-lab/README.md`. LiveKit e voz batch não foram removidos.
