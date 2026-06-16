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
- `backend/internal/services`: reservado para casos de uso e regras de aplicacao.
- `backend/internal/repositories`: reservado para persistencia de dominio.
- `backend/internal/ai`: reservado para provedores de IA.
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
4. `handlers.NewRouter` cria `gin.Engine`.
5. `NewHealthHandler` recebe `database.Postgres`.
6. Servidor HTTP inicia com shutdown gracioso.

### Health Check

1. Cliente chama `GET /health`.
2. `HealthHandler.Check` cria contexto com timeout de 2 segundos.
3. Handler chama `db.Ping`.
4. Se o banco responder, retorna `200` com `status: ok`.
5. Se o banco falhar, retorna `503` com `status: degraded`.

### Frontend

1. `app/page.tsx` renderiza pagina inicial.
2. `HealthPanel` chama `getHealth`.
3. `getHealth` usa `API_INTERNAL_URL` no servidor Next.js.
4. UI exibe status, database e URL publica da API.

### Docker Compose

1. `postgres` sobe com healthcheck.
2. `backend` espera `postgres` ficar healthy.
3. `frontend` depende do `backend` iniciado.

## Dependencias Existentes

Backend diretas:

- Gin para HTTP.
- pgx/pgxpool para PostgreSQL.

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

## Sugestoes Arquiteturais Sem Alteracao de Codigo

- Introduzir interfaces entre handlers, services e repositories antes de implementar conversa, memoria ou IA.
- Evitar que handlers dependam diretamente de provedores externos de IA.
- Definir contratos para `ai`, `memory`, `tools` e `agents` antes da primeira implementacao concreta.
- Adicionar migrations versionadas para entidades reais e decidir se o carregamento via `/docker-entrypoint-initdb.d` sera suficiente ou se sera adotado um migrator dedicado.
- Adicionar testes unitarios para config, database health e handlers.
- Adicionar padrao de erro HTTP antes de ampliar a API.
- Avaliar autenticacao antes de habilitar acesso remoto via Tailscale.
- Manter separacao entre URL interna de container e URL publica de navegador.

## Observacoes de Risco

- A migration atual roda automaticamente apenas quando o volume PostgreSQL e inicializado pela primeira vez.
- O backend falha ao iniciar se o banco nao estiver disponivel.
- Ainda nao ha camada de services/repositories em uso; handlers acessam o wrapper de banco diretamente.
- Nao ha testes automatizados implementados no repositorio.
- `npm audit --omit=dev` aponta vulnerabilidades moderadas em dependencia transitiva do Next.js/PostCSS na versao observada.
