# ARCHITECTURE.md

Documentacao arquitetural do estado atual do Jarvis.

## Estrutura Atual

```text
.
├── backend/
│   ├── cmd/api/
│   ├── docker/
│   ├── internal/
│   │   ├── agents/
│   │   ├── ai/
│   │   ├── config/
│   │   ├── database/
│   │   ├── handlers/
│   │   ├── memory/
│   │   ├── middleware/
│   │   ├── repositories/
│   │   ├── services/
│   │   ├── tools/
│   │   └── voice/
│   ├── migrations/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── app/
│   ├── components/
│   ├── hooks/
│   ├── public/
│   ├── services/
│   ├── types/
│   ├── Dockerfile
│   ├── package.json
│   └── tsconfig.json
├── voice-agent/
│   ├── src/jarvis_voice_agent/
│   ├── tests/
│   ├── pyproject.toml
│   └── README.md
├── docker-compose.yml
├── .env.example
└── README.md
```

## Responsabilidades dos Diretorios

### Backend

- `backend/cmd/api`: ponto de entrada da aplicacao HTTP.
- `backend/internal/config`: leitura de configuracoes via env vars.
- `backend/internal/database`: conexao e pool PostgreSQL.
- `backend/internal/handlers`: handlers HTTP e registro de rotas Gin.
- `backend/internal/middleware`: middlewares HTTP compartilhados; contem CORS.
- `backend/internal/services`: casos de uso e regras de aplicacao; contem chat com tool calling e tools iniciais.
- `backend/internal/repositories`: reservado para persistencia de dominio.
- `backend/internal/ai`: contrato `LLM` e provedores de IA; contem o provider inicial Ollama.
- `backend/internal/memory`: reservado para memoria persistente.
- `backend/internal/tools`: contrato `Tool`, registry, TimeTool, CalculatorTool e SystemTool.
- `backend/internal/voice`: contratos `SpeechRecognizer` e `VoiceProvider`, provider STT por comando, provider STT HTTP local e provider TTS por comando.
- `backend/internal/agents`: reservado para agentes autonomos.
- `backend/internal/voice`: reservado para voz.
- `backend/migrations`: scripts SQL de schema.
- `backend/docker`: Dockerfile do backend.

### Frontend

- `frontend/app`: rotas e layouts do Next.js App Router.
- `frontend/components`: componentes reutilizaveis.
- `frontend/services`: chamadas para APIs.
- `frontend/types`: tipos TypeScript compartilhados.
- `frontend/services/realtime-voice.ts`: cliente do `voice-agent` e do `livekit-client` para criar sessao, entrar na sala e publicar o microfone.
- `frontend/types/realtime-voice.ts`: contrato de sessao LiveKit usado pelo frontend.
- `frontend/hooks`: reservado para hooks.
- `frontend/public`: reservado para assets publicos.

### Voice Agent

- `voice-agent`: fronteira nova para voz realtime, separada do backend Go.
- `voice-agent/src/jarvis_voice_agent/jarvis_core.py`: cliente do Jarvis Core via `POST /api/chat/stream`.
- `voice-agent/src/jarvis_voice_agent/server.py`: servidor local minimo para health check e smoke test de chat.
- `voice-agent/src/jarvis_voice_agent/console.py`: modo console para validar streaming core sem audio.
- `voice-agent/src/jarvis_voice_agent/livekit.py`: emissor de tokens LiveKit para clientes desktop/Android.
- `voice-agent/src/jarvis_voice_agent/realtime.py`: participante LiveKit opcional que assina audio da sala, segmenta fala e chama STT/Jarvis Core.
- `voice-agent/src/jarvis_voice_agent/audio_turn.py`: VAD simples por RMS/silencio para formar turnos de fala.
- `voice-agent/src/jarvis_voice_agent/stt.py`: cliente HTTP multipart para o servidor STT local.
- `voice-agent/src/jarvis_voice_agent/wake.py`: deteccao de wake word por transcricao.

## Fluxos Entre Modulos

### Inicializacao do Backend

1. `cmd/api/main.go` chama `config.Load`.
2. `config.Load` le env vars e aplica defaults.
3. `database.New` cria `pgxpool.Pool` e executa `Ping`.
4. `ai.NewOllamaProvider` cria o provider configurado por `OLLAMA_URL` e `OLLAMA_MODEL`.
5. `services.NewChatService` recebe o contrato `ai.LLM`.
6. `tools.NewRegistry` cria o registry de ferramentas.
7. `TimeTool`, `CalculatorTool` e `SystemTool` sao registradas no registry.
8. `services.NewToolService` recebe o `ToolRegistry`.
9. Providers de STT/TTS sao criados a partir da configuracao; STT pode usar comando local ou servidor HTTP local persistente.
10. `services.NewVoiceService` recebe recognizer, provider e `ChatService`.
11. `handlers.NewRouter` cria `gin.Engine`.
12. `NewHealthHandler` recebe `database.Postgres`.
13. `NewChatHandler` recebe `ChatService`.
14. `NewToolHandler` recebe `ToolService`.
15. `NewVoiceHandler` recebe `VoiceService`.
16. Servidor HTTP inicia com shutdown gracioso.

### Health Check

1. Cliente chama `GET /health`.
2. `HealthHandler.Check` cria contexto com timeout de 2 segundos.
3. Handler chama `db.Ping`.
4. Se o banco responder, retorna `200` com `status: ok`.
5. Se o banco falhar, retorna `503` com `status: degraded`.

### Chat Por Texto Sem Streaming

1. Cliente chama `POST /api/chat` com `{ "message": "..." }`.
2. `ChatHandler` valida JSON, mensagem obrigatoria e tamanho maximo.
3. Handler cria contexto com timeout de 60 segundos.
4. `ChatService` verifica se a mensagem corresponde a uma tool obvia.
5. Se nenhuma tool for necessaria, `ChatService` chama `ai.LLM`.
6. Se uma tool for necessaria, `ChatService` executa a tool pelo `ToolService`.
7. `ToolService` delega a execucao ao `ToolRegistry`.
8. Tools simples retornam respostas deterministicas sem chamada extra ao LLM.
9. Backend retorna `{ "response": "..." }`.
10. Erros de validacao retornam `400`; falhas do LLM retornam `502`.

### Chat Por Texto Com Streaming

1. Cliente chama `POST /api/chat/stream` com `{ "message": "..." }`.
2. `ChatHandler` valida JSON, mensagem obrigatoria e tamanho maximo.
3. Handler responde com `Content-Type: text/event-stream`.
4. Se nenhuma tool for necessaria, `ChatService` chama `LLM.Stream`.
5. Se uma tool for necessaria, `ChatService` envia a resposta deterministica como evento SSE `delta`.
6. Ao terminar, backend envia evento SSE `done`.
7. Falhas durante a geracao viram evento SSE `error`.

### Tools

1. Cliente chama `GET /api/tools`.
2. `ToolHandler.List` chama `ToolService.ListTools`.
3. `ToolService` lista metadados pelo `ToolRegistry`.
4. Backend retorna um array com `name` e `description`.
5. Cliente chama `POST /api/tools/execute` com `{ "tool": "...", "input": "..." }`.
6. `ToolHandler.Execute` valida JSON, nome da ferramenta e tamanho do input.
7. `ToolService.ExecuteTool` delega ao `ToolRegistry`.
8. `ToolRegistry` busca a ferramenta registrada por nome e chama `Execute`.
9. Backend retorna `{ "result": "..." }`.
10. Ferramenta inexistente retorna `404`; input invalido da calculadora retorna `400`.

O fluxo manual de tools permanece disponivel para desenvolvimento e debug. No chat, a decisao automatica de uso de tool acontece no `ChatService` por roteamento local de casos obvios.

### SystemTool

1. O LLM ou o endpoint manual solicita a tool `system`.
2. O input recebido deve ser o nome de uma acao permitida.
3. `SystemTool` busca a acao em uma lista pre-cadastrada.
4. Se a acao nao existir, retorna erro de input invalido.
5. Se existir, executa apenas o comando e argumentos definidos no codigo para aquela acao.

`SystemTool` nao executa comandos arbitrarios, nao interpreta shell e nao usa a entrada do usuario como comando.

### Voz

1. Cliente chama `POST /api/voice/conversation` com multipart `audio`.
2. `VoiceHandler` valida upload, tamanho maximo, idioma e `wakeRequired`.
3. `VoiceService` chama `voice.SpeechRecognizer`.
4. O provider de STT pode chamar um servidor HTTP local persistente ou executar um comando local com arquivo temporario.
5. Se `wakeRequired=true` e a transcricao nao contiver a wake word, o backend retorna `ignored=true` sem chamar LLM, tools ou TTS.
6. Se houver wake word ou se `wakeRequired=false`, a transcricao e enviada para `ChatService`.
7. `ChatService` pode usar LLM e tools pelo `ToolRegistry`.
8. `VoiceService` chama `voice.VoiceProvider` para gerar audio da resposta.
9. O provider local de TTS executa o comando configurado e le o arquivo de audio gerado.
10. Backend retorna transcript, resposta textual, flags de wake word, content type do audio e audio em base64.

Os providers de voz usam processamento local configurado por env vars e nao dependem de APIs pagas por padrao. A troca futura de tecnologia deve implementar os mesmos contratos. O audio bruto nao deve ser persistido por padrao.

### Voice Agent Realtime

1. Cliente desktop ou Android abre uma sessao realtime com o agente de voz.
2. O cliente solicita `POST /livekit/session` no `voice-agent` para receber URL, sala, identidade e token LiveKit.
3. O transporte alvo e WebRTC via LiveKit, com pipeline Pipecat para VAD, STT, LLM e TTS streaming.
4. Apos transcrever uma fala, o agente chama `POST /api/chat/stream` no Jarvis Core.
5. O Jarvis Core continua responsavel por LLM, tools, permissoes, memoria futura e comandos do sistema.
6. O agente transforma os deltas SSE em texto para TTS streaming.
7. O audio de resposta volta pelo transporte realtime.

O slice implementado em `voice-agent` valida a conexao com o Jarvis Core sem audio, por console e HTTP local, emite tokens LiveKit para clientes desktop/Android e possui um participante realtime que assina audio da sala, segmenta turnos, transcreve via STT local e chama o Jarvis Core quando a wake word e detectada. O endpoint batch `POST /api/voice/conversation` permanece como fallback e caminho de debug, nao como arquitetura principal para experiencia natural.

### Frontend

1. `app/page.tsx` renderiza a interface principal de voz.
2. `VoiceAssistant` arma/desarma a escuta por botao.
3. `VoiceAssistant` captura audio com `MediaRecorder`.
4. `VoiceAssistant` usa VAD local com `AnalyserNode` para detectar inicio/fim de fala.
5. `sendVoiceConversation` envia `POST /api/voice/conversation` para `NEXT_PUBLIC_API_URL` com `wakeRequired=true`.
6. Se o backend retornar `ignored=true`, a interface volta ao standby sem resposta falada.
7. `VoiceAssistant` reproduz audio de resposta e anima `Speaking` com volume da reproducao.
6. `ChatPanel` permanece recolhido em `Modo Texto`.
7. `ToolsPanel` permanece recolhido para desenvolvimento e debug.
8. `HealthPanel` chama `getHealth`.
9. `getHealth` usa `API_INTERNAL_URL` no servidor Next.js.

### Docker Compose

1. `postgres` sobe com healthcheck.
2. `backend` espera `postgres` ficar healthy.
3. `frontend` depende do `backend` iniciado.
4. `voice-agent` roda como servico separado e depende do `backend`.
5. `livekit` fica disponivel no profile opcional `realtime` para transporte WebRTC local.

## Dependencias Existentes

Backend diretas:

- Gin para HTTP.
- pgx/pgxpool para PostgreSQL.
- Biblioteca padrao `net/http` para chamadas ao Ollama.
- Biblioteca padrao `os/exec` para providers locais de voz configuraveis.
- Biblioteca padrao `net/http` para provider STT HTTP local.

Voice Agent diretas:

- Python 3.11+.
- Biblioteca padrao para cliente HTTP/SSE e servidor local de smoke test.

Voice Agent opcionais:

- Pipecat para pipeline realtime.
- LiveKit para transporte WebRTC desktop/mobile.

Frontend diretas:

- Next.js.
- React.
- React DOM.

Dev frontend:

- TypeScript.
- Tailwind CSS.
- PostCSS.
- Autoprefixer.
- ESLint e `eslint-config-next`.

Infra:

- Docker Compose.
- PostgreSQL 16.

## Sugestoes Arquiteturais Futuras

- Evoluir contratos entre handlers, services e repositories conforme conversa e memoria crescerem.
- Manter handlers sem dependencia direta de provedores externos de IA.
- Definir contratos para `memory` e `agents` antes da primeira implementacao concreta.
- Manter novas tools atras do contrato `Tool` e do `ToolRegistry`.
- Manter providers de voz batch atras dos contratos `SpeechRecognizer` e `VoiceProvider`.
- Manter voz realtime isolada no `voice-agent`, chamando o Jarvis Core em vez de executar tools diretamente.
- Adicionar migrations versionadas para entidades reais e decidir se o carregamento via `/docker-entrypoint-initdb.d` sera suficiente ou se sera adotado um migrator dedicado.
- Adicionar testes unitarios para config, database health e handlers.
- Adicionar padrao de erro HTTP antes de ampliar muito a API.
- Avaliar autenticacao antes de habilitar acesso remoto via Tailscale.
- Manter separacao entre URL interna de container e URL publica de navegador.

## Observacoes de Risco

- A migration atual roda automaticamente apenas quando o volume PostgreSQL e inicializado pela primeira vez.
- O backend falha ao iniciar se o banco nao estiver disponivel.
- O health check ainda acessa o wrapper de banco diretamente.
- O chat inicial nao persiste historico e nao usa memoria.
- O chat com streaming reduz latencia percebida, mas ainda nao persiste historico e nao implementa cancelamento explicito na UI.
- Tool calling inicial suporta uma chamada de tool por interacao; fluxos multi-step ficam para evolucao futura.
- STT/TTS dependem de binarios e modelos locais configurados no ambiente.
- O endpoint de voz batch ainda usa upload do trecho detectado por VAD; a direcao principal para experiencia natural passa a ser o `voice-agent` realtime.
- Wake word atual e baseada na transcricao, nao em keyword spotting acustico dedicado.
- Ainda ha poucos testes automatizados implementados no repositorio.
- `npm audit --omit=dev` aponta vulnerabilidades moderadas em dependencia transitiva do Next.js/PostCSS na versao observada.
