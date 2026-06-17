# CONTEXT.md

Contexto operacional e conceitual do Jarvis.

## O Que E o Jarvis

Jarvis e um assistente pessoal com objetivo de evoluir para conversacao por texto e voz, memoria persistente, ferramentas e agentes autonomos. A execucao local em Linux, baixo custo operacional e uso inicial de Ollama fazem parte da visao definida.

No estado atual, o projeto possui a fundacao tecnica, uma primeira integracao funcional de conversa por texto via Ollama, uma infraestrutura inicial de tools, uma integracao simples entre LLM e ToolRegistry e uma primeira interface de voz.

## Conceitos Principais

- Assistente: conceito principal planejado. Ainda nao ha entidade de dominio implementada.
- Conversa: existe um fluxo minimo via `POST /api/chat` e `POST /api/chat/stream`, com roteamento local para tools obvias, sem modelo persistido, thread ou historico.
- Memoria: planejada. O diretorio existe, mas nao ha schema ou logica de memoria.
- Ferramentas: implementadas inicialmente com contrato `Tool`, `ToolRegistry`, `TimeTool`, `CalculatorTool`, `SystemTool`, `ToolService`, endpoints de teste e uso automatico pelo chat.
- Agentes: planejados. O diretorio existe, mas nao ha runtime de agentes.
- Voz: implementada inicialmente com contratos `SpeechRecognizer` e `VoiceProvider`, endpoint de conversa por audio e frontend com captura/reproducao. Depende de STT/TTS locais configurados.
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
- `internal/voice`: define contratos e providers locais configuraveis para STT/TTS.
- `migrations`: contem schema inicial carregado pelo container Postgres.
- `docker`: contem Dockerfile do backend.

Frontend:

- `app`: App Router do Next.js com layout e pagina inicial.
- `components`: componentes de UI, atualmente `JarvisInterface`, `VoiceAssistant`, `HealthPanel`, `ChatPanel` e `ToolsPanel`.
- `services`: chamadas para APIs externas, atualmente health check, chat, tools e voice.
- `types`: tipos TypeScript compartilhados para health, chat, tools e voice.
- `hooks`: reservado para hooks futuros.

Infraestrutura:

- `docker-compose.yml`: orquestra PostgreSQL, backend e frontend.
- `.env.example`: documenta variaveis de ambiente.

## Relacionamento Entre Modulos

Fluxo atual:

1. Docker Compose inicia PostgreSQL.
2. PostgreSQL executa migrations montadas em `/docker-entrypoint-initdb.d` na inicializacao do volume.
3. Compose aguarda o healthcheck do PostgreSQL.
4. Backend inicia, carrega env vars e conecta no PostgreSQL.
5. Backend configura `OllamaProvider` com `OLLAMA_URL` e `OLLAMA_MODEL`.
6. Backend registra `GET /health`, `POST /api/chat` e `POST /api/chat/stream`.
7. Backend cria o `ToolRegistry`, registra `TimeTool`, `CalculatorTool` e `SystemTool`, e expoe `GET /api/tools` e `POST /api/tools/execute`.
8. Backend configura providers de voz por comandos locais, como Faster Whisper/Piper ou equivalentes.
9. Frontend inicia e renderiza a pagina inicial como HUD de voz em tela cheia, com modo texto e painel de tools em paineis laterais ocultos por padrao.
10. `HealthPanel` chama `getHealth`.
11. `getHealth` consulta `API_INTERNAL_URL` no servidor Next.js e exibe `NEXT_PUBLIC_API_URL` para o usuario.

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

1. Usuario clica em `Falar com Jarvis`.
2. Frontend captura audio com `MediaRecorder`.
3. A interface mostra estado `Listening` e anima barras com o volume do microfone.
4. Usuario finaliza a fala.
5. Frontend envia `POST /api/voice/conversation` com multipart contendo o audio.
6. `VoiceHandler` valida upload e chama `VoiceService`.
7. `VoiceService` usa `SpeechRecognizer` para transcrever audio em portugues.
8. `VoiceService` envia a transcricao para `ChatService`.
9. `ChatService` pode responder normalmente ou usar tools pelo `ToolRegistry`.
10. `VoiceService` usa `VoiceProvider` para sintetizar a resposta em audio.
11. Frontend reproduz o audio e mostra estado `Speaking`, animando com volume da reproducao.
12. Ao terminar, a interface volta para `Idle`.

Conceito de providers de voz:

- `SpeechRecognizer` e o contrato para STT.
- `VoiceProvider` e o contrato para TTS.
- A implementacao atual usa comandos locais configuraveis por env vars.
- Os comandos sao executados sem shell e com argumentos montados por placeholders controlados.
- A arquitetura permite trocar Faster Whisper, Piper ou equivalentes sem alterar handlers.

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

Nao implementado:

- Memoria de usuario.
- Chamadas multiplas de tools em uma mesma interacao.
- Wake word.
- Streaming bidirecional de audio.
- Agentes.
- Autenticacao.
- Acesso remoto.
