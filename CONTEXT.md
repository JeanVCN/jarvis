# CONTEXT.md

Contexto operacional e conceitual do Jarvis.

## O Que E o Jarvis

Jarvis e um assistente pessoal com objetivo de evoluir para conversacao por texto e voz, memoria persistente, ferramentas e agentes autonomos. A execucao local em Linux, baixo custo operacional e uso inicial de Ollama fazem parte da visao definida.

No estado atual, o projeto possui a fundacao tecnica e uma primeira integracao funcional de conversa por texto via Ollama.

## Conceitos Principais

- Assistente: conceito principal planejado. Ainda nao ha entidade de dominio implementada.
- Conversa: existe um fluxo minimo via `POST /api/chat` e `POST /api/chat/stream`, sem modelo persistido, thread ou historico.
- Memoria: planejada. O diretorio existe, mas nao ha schema ou logica de memoria.
- Ferramentas: planejadas. O diretorio existe, mas nao ha executor ou registry.
- Agentes: planejados. O diretorio existe, mas nao ha runtime de agentes.
- Voz: planejada. O diretorio existe, mas nao ha captura, STT ou TTS.
- IA: implementada inicialmente com contrato `LLM` e `OllamaProvider`.
- Health check: implementado. Verifica backend e conectividade com PostgreSQL.

## Modulos Atuais

Backend:

- `cmd/api`: inicializa configuracao, banco, router HTTP e shutdown gracioso.
- `internal/config`: carrega environment variables e monta endereco HTTP.
- `internal/database`: cria e encapsula pool PostgreSQL.
- `internal/handlers`: registra rotas e implementa `/health`, `POST /api/chat` e `POST /api/chat/stream`.
- `internal/middleware`: contem middlewares HTTP compartilhados, atualmente CORS.
- `internal/ai`: define a interface `LLM` e implementa o provider inicial Ollama.
- `internal/services`: contem o caso de uso inicial de chat.
- `migrations`: contem schema inicial carregado pelo container Postgres.
- `docker`: contem Dockerfile do backend.

Frontend:

- `app`: App Router do Next.js com layout e pagina inicial.
- `components`: componentes de UI, atualmente `HealthPanel` e `ChatPanel`.
- `services`: chamadas para APIs externas, atualmente health check e chat.
- `types`: tipos TypeScript compartilhados para health e chat.
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
7. Frontend inicia e renderiza a pagina inicial com painel de chat.
8. `HealthPanel` chama `getHealth`.
9. `getHealth` consulta `API_INTERNAL_URL` no servidor Next.js e exibe `NEXT_PUBLIC_API_URL` para o usuario.

Fluxo atual de chat:

1. Usuario digita uma mensagem na tela.
2. `ChatPanel` chama `sendChatMessage`.
3. `streamChatMessage` envia `POST /api/chat/stream` para `NEXT_PUBLIC_API_URL`.
4. `ChatHandler` valida a mensagem.
5. `ChatService` chama a interface `LLM`.
6. `OllamaProvider` consulta o Ollama em streaming.
7. Backend transforma chunks do Ollama em eventos SSE.
8. Frontend exibe os deltas em tempo real.

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
- Tela simples de teste de chat.
- Health panel.
- Endpoint `/health`.
- Endpoint `POST /api/chat`.
- Endpoint `POST /api/chat/stream`.
- Verificacao de banco no health check.
- Integracao inicial com Ollama via backend.

Nao implementado:

- Voz.
- Memoria de usuario.
- Ferramentas.
- Agentes.
- Autenticacao.
- Acesso remoto.
