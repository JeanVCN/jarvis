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

Status: aceita.

Contexto:

- O contexto do projeto define Ollama como integracao inicial.
- A fase 2 implementou o primeiro fluxo funcional de chat por texto.

Decisao:

- A primeira integracao de IA deve considerar Ollama antes de OpenAI e Gemini.
- O provider Ollama deve ficar atras da interface interna `LLM`.

Impacto:

- O modulo `ai` suporta um provedor local inicialmente e mantem caminho para provedores remotos no futuro.

### 2026-06-17 - Handlers Usam Services Para Conversa

Status: aceita.

Contexto:

- `POST /api/chat` precisa consultar um LLM sem acoplar regras de aplicacao ao provider Ollama.
- O projeto ja reservava `backend/internal/services` para casos de uso.

Decisao:

- O handler de chat deve depender de `ChatService`.
- `ChatService` deve depender do contrato `ai.LLM`.
- `OllamaProvider` deve ser uma implementacao concreta de `ai.LLM`, configurada no `cmd/api`.

Impacto:

- O endpoint fica protegido de detalhes HTTP da API do Ollama.
- OpenAI, Gemini ou outros provedores podem ser adicionados sem trocar o contrato do handler.
- A camada de services passa a existir com um caso de uso minimo de conversa por texto.

### 2026-06-17 - CORS Local Configuravel

Status: aceita.

Contexto:

- A tela de teste do frontend roda em `localhost:13000`.
- O backend roda em `localhost:18080`.
- Chamadas client-side do navegador para `/api/chat` exigem permissao CORS.
- Em alguns ambientes, `localhost` pode adicionar atraso por resolucao IPv6/IPv4.

Decisao:

- O backend deve configurar CORS por `CORS_ALLOWED_ORIGIN`, com default local para desenvolvimento.
- O default local deve aceitar `http://127.0.0.1:13000` e `http://localhost:13000`.
- O middleware de CORS deve ficar em `backend/internal/middleware`, fora de `handlers`.
- URLs publicas locais do frontend devem preferir `127.0.0.1`.

Impacto:

- A tela de teste pode chamar o backend diretamente do navegador.
- Ambientes futuros podem ajustar a origem permitida sem alterar codigo.
- `handlers` permanece focado em rotas e handlers HTTP.
- O caminho local evita atrasos causados por resolucao de `localhost` quando o ambiente tenta IPv6 antes de IPv4.

### 2026-06-17 - Portas Padrao Dedicadas Para Desenvolvimento

Status: aceita.

Contexto:

- Portas `8080` e `3000` sao comuns e podem estar ocupadas por outros projetos locais.
- O Jarvis precisa ter defaults consistentes entre env vars, backend, frontend, Docker Compose e VS Code.

Decisao:

- Usar `18080` como porta padrao do backend.
- Usar `13000` como porta padrao do frontend.
- Manter `5432` para PostgreSQL e `11434` para Ollama.

Impacto:

- Reduz conflito com outros projetos em desenvolvimento.
- O F5 do VS Code, Docker Compose e fallbacks de configuracao passam a usar as mesmas portas.

### 2026-06-17 - Streaming De Chat Via SSE

Status: aceita.

Contexto:

- O endpoint inicial `POST /api/chat` usava `stream: false` no Ollama.
- Respostas longas so chegavam ao frontend depois da geracao completa.
- O Ollama entrega chunks em tempo real quando streaming esta ativo.

Decisao:

- Manter `POST /api/chat` como endpoint simples sem streaming.
- Criar `POST /api/chat/stream` usando `text/event-stream`.
- Usar SSE para transmitir eventos `delta`, `done` e `error`.
- O frontend deve consumir o stream com `fetch`, pois `EventSource` nao suporta `POST` com JSON.

Impacto:

- O usuario passa a ver a resposta surgir em tempo real.
- A latencia percebida fica menor sem exigir WebSocket nesta fase.
- WebSocket permanece como opcao futura para fluxos bidirecionais mais complexos, como voz, tools ou agentes.

## Pendentes de Validacao

### Estrategia de Migracoes

Status: pendente de validacao.

Hipotese:

- A migration via `/docker-entrypoint-initdb.d` pode ser suficiente na fundacao, mas talvez nao baste para evolucao incremental em ambientes existentes.

### Contratos Entre Services, Repositories e Providers de IA

Status: parcialmente validado.

Hipotese:

- O contrato inicial `ai.LLM` atende ao primeiro fluxo de conversa, mas pode evoluir quando houver historico, streaming, tools ou memoria.

### Estrategia de Autenticacao

Status: pendente de validacao.

Hipotese:

- Acesso remoto via Tailscale exigira autenticacao ou controles equivalentes.

### Estrategia de Memoria Persistente

Status: pendente de validacao.

Hipotese:

- Memoria pode exigir separacao entre historico conversacional, fatos persistentes e configuracoes do usuario.
