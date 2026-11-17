# CONTEXT.md

Contexto operacional e conceitual do Jarvis.

## Contexto vigente — Jarvis pessoal no WhatsApp

A prioridade é validar texto e delegação ao OpenCode no chat **Você** do
próprio número. O código já tem pareamento separado por conta, GPT autenticado
para conversa normal e `/code` persistente para tarefas no OpenCode. O bridge
fica em loopback. A interface web continua sendo uma alternativa de acesso
privado ao workspace, com validação separada. A ordem de trabalho, o piloto
pendente, o histórico e o plano de voz estão no início de [ROADMAP.md](ROADMAP.md).

## Contexto complementar — interface web Jarvis ↔ OpenCode

Planejamento e critérios da interface web estão em `ROADMAP.md`, fases C0–C4.

- Objetivo ativo: programar pelo celular usando Jarvis como interface e ponto de
  controle, delegando tarefas de código ao OpenCode V2 que roda no computador dos
  projetos. Jarvis exibe progresso, resultado/diff e permite continuar sessões.
- Decisão: integrar OpenCode, não construir agora um harness próprio completo.
  `opencode-bridge` e tela `/code` agora implementam leitura, criação/continuação
  de sessão, prompts, respostas/diffs limitados e aprovações `once`/`reject`.
  Escrita exige `JARVIS_OPENCODE_WRITE_ENABLED=true` no Core e no bridge e fica
  desligada por padrão. O fluxo usa allowlist, token interno, bind loopback e
  journal PostgreSQL; Compose não habilita o bridge. Mocks/builds passaram, mas
  serviço OpenCode real e implantação privada ainda não foram validados.
- Jarvis já tem Core Go, frontend Next.js, agente de voz Python e `/api/v2` com
  histórico/eventos/journal. A API permite somente `time` e `calculator`; não
  ampliar essa autorização implicitamente para comandos do computador.
- Não haverá login Jarvis. O frontend não tem tela/rotas de login e valida a
  origem das mutações. A fronteira de acesso é a máquina local/Tailscale; Core,
  bridge e OpenCode devem continuar em loopback, com Tailscale permitindo somente
  dispositivos autorizados. Regras por dispositivo e proxy privado ainda
  precisam de validação prática. Não expor Core/OpenCode diretamente nem aceitar
  diretórios arbitrários. Um futuro app Android exigirá pareamento por instalação
  e revogação, sem criar conta Jarvis.
- Eventos OpenCode não têm replay/reconexão automática pelo client; após falha,
  Jarvis deve consultar o estado da sessão antes de atualizar ou repetir pedidos.
  Shell roda com autoridade do usuário local; aprovações não são sandbox.
- Contrato e pendências de validação OpenCode V2 estão em
  `docs/opencode-integration.md`: `session.prompt` retorna item inbox e a
  semântica do campo `id`, permissões efetivas e reconciliação ainda precisam de
  confirmação com serviço real. Nunca repetir automaticamente operação incerta.
- Automação residencial Google Home está pausada até haver Android. Probe
  `android/home-probe` compilado permanece como checkpoint; OAuth e descoberta
  não foram feitos. Evidências em `docs/home-apis/README.md`.
- Próxima ação é configurar/validar acesso privado por dispositivo e qualificar
  a escrita com OpenCode real em projeto descartável antes de habilitar o flag.
  Fases e critérios em `ROADMAP.md`; arquitetura em `ARCHITECTURE.md`.
- Avaliação em 2026-11-16: estamos perto de uma prova piloto, não de uso remoto
  pronto. O código read/write existe atrás de flag; faltam ACL Tailscale testada,
  validação real OpenCode e ciclo pelo telefone. IoT pode voltar à prioridade após
  esse gate, mas integração física continua bloqueada até Android; hoje há somente
  probe read-only sem OAuth/discovery real.

## Contexto das fases anteriores

## O Que E o Jarvis

Jarvis e um assistente pessoal com objetivo de evoluir para conversacao por texto e voz, memoria persistente, ferramentas e agentes autonomos. A execucao local em Linux, baixo custo operacional e uso inicial de Ollama fazem parte da visao definida.

No estado atual, o projeto possui a fundacao tecnica, uma primeira integracao funcional de conversa por texto via Ollama, uma infraestrutura inicial de tools, uma integracao simples entre LLM e ToolRegistry, uma primeira interface de voz batch e a base de um `voice-agent` separado para evoluir voz realtime.

## Conceitos Principais

- Assistente: conceito principal planejado. Ainda nao ha entidade de dominio implementada.
- Conversa: existe um fluxo minimo via `POST /api/chat` e `POST /api/chat/stream`, com roteamento local para tools obvias, sem modelo persistido, thread ou historico.
- Memoria: planejada. O diretorio existe, mas nao ha schema ou logica de memoria.
- Ferramentas: implementadas inicialmente com contrato `Tool`, `ToolRegistry`, `TimeTool`, `CalculatorTool`, `SystemTool`, `ToolService`, endpoints de teste e uso automatico pelo chat.
- Agentes: planejados. O diretorio existe, mas nao ha runtime de agentes.
- Voz: implementada com contratos `SpeechRecognizer` e `VoiceProvider`, endpoint de conversa por audio, frontend com VAD local/wake word e STT/TTS locais configurados; este fluxo fica como fallback batch.
- Voice Agent: modulo Python separado para migrar a experiencia principal de voz para realtime, chamando o Jarvis Core por `POST /api/chat/stream`.
- IA: implementada inicialmente com contrato `LLM` e `OllamaProvider`.
- Health check: implementado. Verifica backend e conectividade com PostgreSQL.

## Modulos Atuais

Backend:

- `cmd/api`: inicializa configuracao, banco, router HTTP e shutdown gracioso.
- `internal/config`: carrega environment variables e monta endereco HTTP.
- `internal/database`: cria e encapsula pool PostgreSQL.
- `internal/handlers`: registra rotas e implementa `/health`, `POST /api/chat`, `POST /api/chat/stream`, `GET /api/tools`, `POST /api/tools/execute` e `POST /api/voice/conversation`.
- `internal/middleware`: contem middlewares HTTP compartilhados, atualmente CORS.
- `internal/ai`: define a interface `LLM` e implementa o provider inicial Ollama.
- `internal/services`: contem os casos de uso iniciais de chat, roteamento local para tools obvias e services de tools/voz.
- `internal/tools`: define o contrato `Tool`, o `ToolRegistry`, `TimeTool`, `CalculatorTool` e `SystemTool`.
- `internal/voice`: define contratos e providers locais configuraveis para STT/TTS, incluindo STT por comando e STT por servidor HTTP local persistente.
- `migrations`: contem schema inicial carregado pelo container Postgres.
- `docker`: contem Dockerfile do backend.

Frontend:

- `app`: App Router do Next.js com layout e pagina inicial.
- `components`: componentes de UI, atualmente `JarvisInterface`, `VoiceAssistant`, `HealthPanel`, `ChatPanel` e `ToolsPanel`.
- `services`: chamadas para APIs externas, atualmente health check, chat, tools e voice.
- `types`: tipos TypeScript compartilhados para health, chat, tools e voice.
- `hooks`: reservado para hooks futuros.
- `services/realtime-voice.ts`: cria sessao realtime no `voice-agent` usando `NEXT_PUBLIC_VOICE_AGENT_URL`.
- `types/realtime-voice.ts`: tipos da sessao LiveKit retornada pelo `voice-agent`.

Infraestrutura:

- `docker-compose.yml`: orquestra PostgreSQL, backend, frontend e `voice-agent`.
- `livekit`: servidor WebRTC local opcional via profile Compose `realtime`.
- `.env.example`: documenta variaveis de ambiente.

Voice Agent:

- `voice-agent/src/jarvis_voice_agent/jarvis_core.py`: cliente SSE para o Jarvis Core.
- `voice-agent/src/jarvis_voice_agent/server.py`: health check e smoke test HTTP local.
- `voice-agent/src/jarvis_voice_agent/console.py`: console para validar streaming sem audio.
- `voice-agent/src/jarvis_voice_agent/livekit.py`: emissor local de tokens LiveKit.
- `voice-agent/src/jarvis_voice_agent/realtime.py`: participante LiveKit opcional que entra na sala, assina faixas de audio, segmenta turnos e aciona STT/Jarvis Core.
- `voice-agent/src/jarvis_voice_agent/audio_turn.py`: detector simples de turno de fala por RMS/silencio.
- `voice-agent/src/jarvis_voice_agent/stt.py`: cliente HTTP multipart para o servidor STT local.
- `voice-agent/src/jarvis_voice_agent/wake.py`: normalizacao e deteccao de wake word por transcricao.

## Relacionamento Entre Modulos

Fluxo atual:

1. Docker Compose inicia PostgreSQL.
2. PostgreSQL executa migrations montadas em `/docker-entrypoint-initdb.d` na inicializacao do volume.
3. Compose aguarda o healthcheck do PostgreSQL.
4. Backend inicia, carrega env vars e conecta no PostgreSQL.
5. Backend configura `OllamaProvider` com `OLLAMA_URL` e `OLLAMA_MODEL`.
6. Backend registra `GET /health`, `POST /api/chat` e `POST /api/chat/stream`.
7. Backend cria o `ToolRegistry`, registra `TimeTool`, `CalculatorTool` e `SystemTool`, e expoe `GET /api/tools` e `POST /api/tools/execute`.
8. Backend configura providers de voz locais, como servidor STT persistente, comandos STT/TTS, Faster Whisper, Sherpa-ONNX, Piper ou equivalentes.
9. Voice Agent pode iniciar como servico separado, apontando para o backend em `JARVIS_CORE_URL`.
10. Frontend inicia e renderiza a pagina inicial como HUD de voz em tela cheia, com modo texto e painel de tools em paineis laterais ocultos por padrao.
11. `HealthPanel` chama `getHealth`.
12. `getHealth` consulta `API_INTERNAL_URL` no servidor Next.js e exibe `NEXT_PUBLIC_API_URL` para o usuario.

Fluxo atual de chat:

1. Usuario digita uma mensagem na tela.
2. `ChatPanel` chama `sendChatMessage`.
3. `streamChatMessage` envia `POST /api/chat/stream` para `NEXT_PUBLIC_API_URL`.
4. `ChatHandler` valida a mensagem.
5. `ChatService` verifica localmente se a mensagem corresponde a uma tool obvia, como hora/data, calculo basico ou acao controlada do sistema.
6. Se nenhuma tool for necessaria, `ChatService` chama `LLM.Stream` e o backend envia os deltas em tempo real.
7. Se uma tool simples for necessaria, `ChatService` executa a ferramenta pelo `ToolRegistry`.
8. `ChatService` retorna uma resposta curta e deterministica para evitar latencia e contradicoes do LLM.
9. Frontend exibe a resposta.

Fluxo atual de tools:

1. Frontend chama `GET /api/tools` para listar ferramentas disponiveis.
2. `ToolHandler` chama `ToolService.ListTools`.
3. `ToolService` consulta o `ToolRegistry`.
4. Frontend envia `POST /api/tools/execute` com `tool` e `input`.
5. `ToolHandler` valida JSON, ferramenta obrigatoria e tamanho maximo do input.
6. `ToolService` executa a ferramenta pelo `ToolRegistry`.
7. O `ToolRegistry` localiza a implementacao concreta por nome.
8. A tool executa e retorna resultado textual.
9. Backend retorna `{ "result": "..." }`.

Conceito de `ToolRegistry`:

- O `ToolRegistry` e o ponto central de descoberta e execucao de ferramentas.
- Services, handlers, agentes ou fluxos futuros de IA devem falar com o registry, nao com implementacoes concretas.
- Novas tools devem implementar a interface `Tool` e ser registradas na inicializacao do backend.
- A integracao automatica atual ocorre no `ChatService`, que usa um roteador local para casos obvios e executa tools pelo registry antes de responder.
- A execucao automatica de tools sempre passa pelo `ToolRegistry`.
- Uma camada futura de planejamento por LLM pode voltar a solicitar tools por JSON, desde que preserve o streaming direto para mensagens normais.

Conceito de `SystemTool`:

- `SystemTool` executa apenas acoes pre-cadastradas.
- A entrada do usuario ou do LLM e tratada como nome de acao, nao como comando.
- Nao ha execucao livre de shell.
- Acoes iniciais: `open_browser`, `open_vscode`, `open_terminal`, `open_spotify` e `lock_screen`.
- Novas acoes devem ser adicionadas explicitamente na configuracao da tool.

Fluxo atual de voz:

1. Usuario arma a escuta por botao.
2. Frontend captura audio com `MediaRecorder` e usa VAD local pelo `AnalyserNode`.
3. A interface mostra estado `Listening` enquanto aguarda a chamada.
4. Quando detecta fala, o frontend grava apenas o trecho falado e encerra automaticamente apos silencio.
5. Frontend envia `POST /api/voice/conversation` com multipart contendo o trecho de audio e `wakeRequired=true`.
6. `VoiceHandler` valida upload e chama `VoiceService`.
7. `VoiceService` usa `SpeechRecognizer` para transcrever audio em portugues.
8. Se a transcricao nao contiver wake word, o backend retorna `ignored=true` sem chamar LLM, tools ou TTS.
9. Se a wake word for detectada, `VoiceService` remove a wake word e envia o comando para `ChatService`.
10. `ChatService` pode responder normalmente ou usar tools pelo `ToolRegistry`.
11. `VoiceService` usa `VoiceProvider` para sintetizar a resposta em audio.
12. Frontend reproduz o audio e mostra estado `Speaking`, animando com volume da reproducao.
13. Ao terminar, a interface volta ao standby se a escuta continuar armada.

Conceito de providers de voz:

- `SpeechRecognizer` e o contrato para STT.
- `VoiceProvider` e o contrato para TTS.
- A implementacao atual suporta comandos locais configuraveis por env vars e servidor STT HTTP local persistente.
- Os comandos sao executados sem shell e com argumentos montados por placeholders controlados.
- A arquitetura permite trocar Faster Whisper, Sherpa-ONNX, Piper ou equivalentes sem alterar handlers.
- O modo de privacidade esperado e local: audio bruto deve ser temporario e descartado apos a transcricao.

Fluxo planejado de voz realtime:

1. Cliente desktop ou Android abre uma sessao realtime com o `voice-agent`.
2. O cliente solicita `POST /livekit/session` para receber token de entrada na sala LiveKit.
3. O transporte alvo e LiveKit WebRTC com pipeline Pipecat.
4. O agente usa VAD/turn detection, STT streaming e TTS streaming.
5. Para cada comando transcrito, o agente chama `POST /api/chat/stream` no Jarvis Core.
6. O backend Go decide LLM, tools, permissoes, memoria futura e comandos locais.
7. O agente converte deltas de texto em audio e envia de volta pelo transporte realtime.

O agente Python nao deve executar comandos locais diretamente; ele e uma fronteira de audio e transporte.

Fluxo planejado de alto nivel:

1. Usuario acessa o Jarvis via navegador.
2. Usuario interage por texto ou voz.
3. Backend recebe a solicitacao.
4. Camada de services coordena IA, memoria, ferramentas e agentes.
5. Repositories persistem dados no PostgreSQL.
6. Provedores de IA respondem por interfaces internas.
7. Frontend exibe resposta e estado da conversa.

## Estado Atual do Produto

Implementado:

- Tela inicial.
- Tela principal baseada em voz com HUD futurista e nucleo central animado.
- Tela simples de teste de chat.
- Tela simples de teste de tools.
- Health panel.
- Endpoint `/health`.
- Endpoint `POST /api/chat`.
- Endpoint `POST /api/chat/stream`.
- Endpoint `GET /api/tools`.
- Endpoint `POST /api/tools/execute`.
- Endpoint `POST /api/voice/conversation`.
- Verificacao de banco no health check.
- Integracao inicial com Ollama via backend.
- `ToolRegistry` com `TimeTool`, `CalculatorTool` e `SystemTool`.
- Escolha automatica de tools obvias por roteamento local, sem chamada extra ao LLM para tools simples.
- Captura de audio no frontend e resposta falada via provider TTS local configurado.
- VAD local no frontend para detectar fim de fala sem clique.
- Wake word simples por transcricao para evitar executar comandos sem chamada ao Jarvis.
- Servidor STT local persistente para manter modelo carregado entre falas.
- Base `voice-agent` com cliente do Jarvis Core, servidor local e console de smoke test.
- Endpoint `POST /livekit/session` no `voice-agent` para emitir token de sala LiveKit.
- HUD principal do frontend consegue criar sessao realtime, entrar na sala LiveKit e publicar o microfone.
- Participante realtime do `voice-agent` consegue entrar na mesma sala, segmentar audio, transcrever via STT local e chamar o Jarvis Core apos wake word.

Nao implementado:

- Memoria de usuario.
- Chamadas multiplas de tools em uma mesma interacao.
- Wake word acustica dedicada.
- Transporte LiveKit/Pipecat com audio realtime conectado.
- APK Android.
- Agentes.
- Autenticacao.
- Acesso remoto.

## Sessões de voz contínua e comparação de perfis

O laboratório `/lab` separa sessão de transporte, conversa persistida e turno.
Um turno identifica entrada, deltas gerados, ferramentas, áudio enviado e progresso
estimado de reprodução. Interromper invalida a geração anterior e preserva a captura
nova; não desfaz ferramentas já executadas. Reconectar conserva conversa, não
reproduz áudio antigo nem repete automaticamente ações. Provedores de IA e de voz
são escolhas independentes, e o envio de áudio remoto não faz parte do perfil padrão.
