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
- `backend/internal/services`: casos de uso e regras de aplicacao; contem o chat inicial.
- `backend/internal/repositories`: reservado para persistencia de dominio.
- `backend/internal/ai`: contrato `LLM` e provedores de IA; contem o provider inicial Ollama.
- `backend/internal/memory`: reservado para memoria persistente.
- `backend/internal/tools`: reservado para ferramentas.
- `backend/internal/agents`: reservado para agentes autonomos.
- `backend/internal/voice`: reservado para voz.
- `backend/migrations`: scripts SQL de schema.
- `backend/docker`: Dockerfile do backend.

### Frontend

- `frontend/app`: rotas e layouts do Next.js App Router.
- `frontend/components`: componentes reutilizaveis.
- `frontend/services`: chamadas para APIs.
- `frontend/types`: tipos TypeScript compartilhados.
- `frontend/hooks`: reservado para hooks.
- `frontend/public`: reservado para assets publicos.

## Fluxos Entre Modulos

### Inicializacao do Backend

1. `cmd/api/main.go` chama `config.Load`.
2. `config.Load` le env vars e aplica defaults.
3. `database.New` cria `pgxpool.Pool` e executa `Ping`.
4. `ai.NewOllamaProvider` cria o provider configurado por `OLLAMA_URL` e `OLLAMA_MODEL`.
5. `services.NewChatService` recebe o contrato `ai.LLM`.
6. `handlers.NewRouter` cria `gin.Engine`.
7. `NewHealthHandler` recebe `database.Postgres`.
8. `NewChatHandler` recebe `ChatService`.
9. Servidor HTTP inicia com shutdown gracioso.

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
4. `ChatService` recebe a mensagem e chama `ai.LLM`.
5. `OllamaProvider` envia a requisicao para `/api/generate` do Ollama com `stream: false`.
6. Backend retorna `200` com `{ "response": "..." }`.
7. Erros de validacao retornam `400`; falhas do LLM retornam `502`.

### Chat Por Texto Com Streaming

1. Cliente chama `POST /api/chat/stream` com `{ "message": "..." }`.
2. `ChatHandler` valida JSON, mensagem obrigatoria e tamanho maximo.
3. Handler responde com `Content-Type: text/event-stream`.
4. `ChatService` chama `ai.LLM.Stream`.
5. `OllamaProvider` envia requisicao para `/api/generate` do Ollama com `stream: true`.
6. Cada chunk NDJSON do Ollama vira evento SSE `delta`.
7. Ao terminar, backend envia evento SSE `done`.
8. Falhas durante a geracao viram evento SSE `error`.

### Frontend

1. `app/page.tsx` renderiza pagina inicial.
2. `ChatPanel` renderiza o formulario de teste de chat.
3. `streamChatMessage` envia `POST /api/chat/stream` para `NEXT_PUBLIC_API_URL`.
4. `HealthPanel` chama `getHealth`.
5. `getHealth` usa `API_INTERNAL_URL` no servidor Next.js.
6. UI exibe resposta do chat, status, database e URL publica da API.

### Docker Compose

1. `postgres` sobe com healthcheck.
2. `backend` espera `postgres` ficar healthy.
3. `frontend` depende do `backend` iniciado.

## Dependencias Existentes

Backend diretas:

- Gin para HTTP.
- pgx/pgxpool para PostgreSQL.
- Biblioteca padrao `net/http` para chamadas ao Ollama.

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
- Definir contratos para `memory`, `tools` e `agents` antes da primeira implementacao concreta.
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
- Nao ha testes automatizados implementados no repositorio.
- `npm audit --omit=dev` aponta vulnerabilidades moderadas em dependencia transitiva do Next.js/PostCSS na versao observada.
