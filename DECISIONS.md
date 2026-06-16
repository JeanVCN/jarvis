# DECISIONS.md

Registro de decisoes arquiteturais do Jarvis.

Use este arquivo como ADR leve. Novas decisoes relevantes devem ser adicionadas com data, contexto e impacto.

## Decisoes Observadas

### 2026-06-16 - Monorepo com Backend e Frontend Separados

Status: aceita.

Contexto:

- O repositorio contem `backend/` e `frontend/` na raiz.

Decisao:

- Manter backend e frontend no mesmo repositorio, com responsabilidades separadas por diretorio.

Impacto:

- Facilita orquestracao por Docker Compose.
- Permite evolucao coordenada de contratos API/UI.

### 2026-06-16 - Backend em Go com Gin

Status: aceita.

Contexto:

- `backend/go.mod` usa Go 1.25.
- O router e os handlers usam Gin.

Decisao:

- Usar Go e Gin como base HTTP do backend.

Impacto:

- API simples e performatica para evoluir em modulos internos.

### 2026-06-16 - PostgreSQL como Banco Principal

Status: aceita.

Contexto:

- Docker Compose inclui `postgres:16-alpine`.
- Backend usa `pgx/v5`.
- Health check valida conectividade com PostgreSQL.

Decisao:

- Usar PostgreSQL para persistencia.

Impacto:

- Memoria persistente e historico futuro devem considerar PostgreSQL como base inicial.

### 2026-06-16 - Configuracao por Environment Variables

Status: aceita.

Contexto:

- `.env.example` lista variaveis de configuracao.
- `internal/config` carrega env vars com defaults.

Decisao:

- Configuracoes operacionais devem vir de env vars.

Impacto:

- Facilita execucao local, Docker Compose e futuro deploy.

### 2026-06-16 - Docker Compose como Orquestracao Local

Status: aceita.

Contexto:

- `docker-compose.yml` orquestra `postgres`, `backend` e `frontend`.

Decisao:

- Usar Docker Compose para ambiente local integrado.

Impacto:

- Novos colaboradores e agentes podem subir a fundacao com um comando.

### 2026-06-16 - Health Check Verifica Banco

Status: aceita.

Contexto:

- `GET /health` chama `db.Ping`.

Decisao:

- Health check deve refletir tambem disponibilidade do PostgreSQL.

Impacto:

- Falhas de banco aparecem como `503` e `status: degraded`.

### 2026-06-16 - Separacao de URL Interna e Publica da API

Status: aceita.

Contexto:

- Frontend usa `API_INTERNAL_URL` para chamadas server-side.
- Frontend usa `NEXT_PUBLIC_API_URL` para representar URL publica ao usuario.

Decisao:

- Manter URLs separadas para resolver diferenca entre rede Docker e navegador.

Impacto:

- Evita que server components tentem acessar `localhost` incorreto dentro do container.

### 2026-06-16 - Build do Next.js com Webpack

Status: aceita.

Contexto:

- `frontend/package.json` define `next build --webpack`.

Decisao:

- Usar webpack no build de producao do Next.js.

Impacto:

- Evita instabilidade observada com Turbopack no ambiente atual.

### 2026-06-16 - Diretorios Futuros Criados Sem Implementacao

Status: aceita.

Contexto:

- Existem diretorios `ai`, `voice`, `memory`, `tools`, `agents`, `services` e `repositories`.

Decisao:

- Preservar limites arquiteturais para evolucao futura sem implementar esses modulos na fundacao.

Impacto:

- Facilita distribuicao futura de trabalho.
- Evita acoplamento prematuro.

## Decisoes Explicitamente Definidas Pelo Projeto

### 2026-06-16 - Nao Implementar IA, Voz ou Agentes na Fase Inicial

Status: aceita.

Contexto:

- A fase inicial solicitou foco exclusivo na fundacao.

Decisao:

- IA, voz e agentes devem permanecer planejados ate fases futuras.

Impacto:

- Mudancas nesses modulos devem esperar definicao de contratos e roadmap.

### 2026-06-16 - Ollama Como Integracao Inicial de IA

Status: planejada.

Contexto:

- O contexto do projeto define Ollama como integracao inicial.

Decisao:

- A primeira integracao de IA deve considerar Ollama antes de OpenAI e Gemini.

Impacto:

- O modulo `ai` deve ser desenhado para suportar provedor local inicialmente e provedores remotos no futuro.

## Pendentes de Validacao

### Estrategia de Migracoes

Status: pendente de validacao.

Hipotese:

- A migration via `/docker-entrypoint-initdb.d` pode ser suficiente na fundacao, mas talvez nao baste para evolucao incremental em ambientes existentes.

### Contratos Entre Services, Repositories e Providers de IA

Status: pendente de validacao.

Hipotese:

- Interfaces devem ser definidas antes de implementar conversa, memoria e IA.

### Estrategia de Autenticacao

Status: pendente de validacao.

Hipotese:

- Acesso remoto via Tailscale exigira autenticacao ou controles equivalentes.

### Estrategia de Memoria Persistente

Status: pendente de validacao.

Hipotese:

- Memoria pode exigir separacao entre historico conversacional, fatos persistentes e configuracoes do usuario.
