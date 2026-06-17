# BRAIN.md

Memoria permanente do projeto Jarvis.

Este arquivo deve preservar o entendimento de longo prazo do projeto. Ele deve registrar apenas informacoes observadas no repositorio ou explicitamente definidas pelos mantenedores.

## Objetivo do Jarvis

Jarvis e um assistente pessoal inspirado no conceito do JARVIS do Homem de Ferro.

Objetivos explicitamente definidos:

- Conversacao por texto.
- Conversacao por voz.
- Memoria persistente.
- Sistema de ferramentas.
- Sistema de agentes autonomos.
- Execucao local em Linux.
- Baixo custo operacional.
- Integracao inicial com Ollama.
- Acesso via navegador desktop e celular.
- Possibilidade de acesso remoto via Tailscale.

## Visao de Longo Prazo

O projeto deve evoluir gradualmente sem exigir reescrita futura. A base atual prioriza separacao de responsabilidades, configuracao por environment variables e preparacao para integrar IA, voz, memoria, ferramentas e agentes em fases posteriores.

Suporte de IA explicitamente planejado:

- Ollama inicialmente.
- OpenAI no futuro.
- Gemini no futuro.

## Tecnologias Utilizadas

Backend observado:

- Go 1.25, conforme `backend/go.mod`.
- Gin `v1.10.0`.
- PostgreSQL via `pgx/v5`.
- Docker com imagem `golang:1.25-alpine` para build e `alpine:3.20` para runtime.

Frontend observado:

- Next.js `16.2.9`.
- React `19.2.1`.
- TypeScript `5.6.3`.
- Tailwind CSS `3.4.13`.
- ESLint `9.39.4`.
- Build Next.js usando `next build --webpack`.

Infraestrutura observada:

- Docker Compose.
- PostgreSQL `16-alpine`.
- Volume Docker `postgres_data`.
- Migration inicial montada em `/docker-entrypoint-initdb.d`.

## Escolhas Arquiteturais Identificadas

- Monorepo com `backend/` e `frontend/`.
- Backend organizado em `cmd/` e `internal/`.
- `cmd/api` contem o ponto de entrada HTTP.
- Configuracao do backend centralizada em `internal/config`.
- Conexao PostgreSQL encapsulada em `internal/database`.
- Rotas HTTP ficam em `internal/handlers`.
- Middlewares HTTP compartilhados ficam em `backend/internal/middleware`.
- `backend/internal/ai` contem o contrato `LLM` e a implementacao inicial `OllamaProvider`.
- `backend/internal/services` contem o caso de uso inicial de chat por texto.
- Diretorios `voice`, `memory`, `tools`, `agents` e `repositories` seguem como limites arquiteturais para evolucao futura.
- Frontend usa App Router do Next.js em `frontend/app`.
- Frontend separa componentes, services, hooks e types.
- O frontend consulta o backend pelo service `frontend/services/health.ts`.
- O frontend possui service de chat para chamar `POST /api/chat`.
- O frontend usa `POST /api/chat/stream` para renderizar respostas do chat em tempo real via SSE.
- URLs internas e publicas da API sao separadas por `API_INTERNAL_URL` e `NEXT_PUBLIC_API_URL`.

## Restricoes do Projeto

Restricoes explicitamente definidas:

- Nao implementar IA antes da fase apropriada.
- Nao implementar voz antes da fase apropriada.
- Nao implementar agentes antes da fase apropriada.
- Evoluir gradualmente, sem reescrita futura.
- Manter baixo custo operacional.
- Priorizar execucao local em Linux.

Restricoes observadas:

- Configuracoes devem vir de environment variables.
- O backend depende de PostgreSQL disponivel ao iniciar.
- A integracao Ollama depende de `OLLAMA_URL` e `OLLAMA_MODEL`.
- O health check verifica conectividade real com o banco.
- O Compose espera que o Postgres esteja healthy antes de iniciar o backend.

## Filosofia de Desenvolvimento

- A estrutura deve preparar o projeto para crescimento incremental.
- Modulos ainda nao implementados devem ser preservados como fronteiras, nao preenchidos artificialmente.
- Decisoes relevantes devem ser documentadas antes que se tornem conhecimento implicito.
- O repositorio atual e a fonte primaria da verdade.
- Evitar reestruturacoes sem necessidade.
- Favorecer clareza e baixo acoplamento entre backend, frontend, banco e futuros modulos de IA.

## Padroes Encontrados no Codigo

Backend:

- Pacotes pequenos com responsabilidade direta.
- Construtores nomeados, como `New`, `NewRouter` e `NewHealthHandler`.
- Uso de `context.Context` para operacoes com banco.
- Timeouts explicitos em conexao, ping e shutdown.
- Shutdown gracioso via `SIGINT` e `SIGTERM`.
- Respostas JSON em handlers Gin.
- Fallbacks de environment variables por funcao auxiliar `getEnv`.
- Provedores de IA devem ficar atras da interface `ai.LLM`.
- Handlers de chat devem chamar services, nao provedores externos diretamente.
- Streaming de chat usa SSE como caminho inicial; WebSocket fica reservado para fluxos bidirecionais futuros.

Frontend:

- Componentes React assíncronos no App Router para server-side e componentes client-side para interacoes.
- Services centralizados para chamadas externas.
- Types compartilhados em `frontend/types`.
- Alias TypeScript `@/*`.
- Tailwind com tema escuro simples e cores customizadas `surface`, `panel` e `accent`.

Docker:

- Builds multi-stage para backend e frontend.
- Usuarios nao-root nos containers finais.
- `.dockerignore` usado para reduzir contexto de build.
