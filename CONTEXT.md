# CONTEXT.md

Contexto operacional e conceitual do Jarvis.

## O Que E o Jarvis

Jarvis e um assistente pessoal com objetivo de evoluir para conversacao por texto e voz, memoria persistente, ferramentas e agentes autonomos. A execucao local em Linux, baixo custo operacional e uso inicial de Ollama fazem parte da visao definida.

No estado atual, o projeto e uma fundacao tecnica: backend, frontend, banco, Docker Compose e health check.

## Conceitos Principais

- Assistente: conceito principal planejado. Ainda nao ha entidade de dominio implementada.
- Conversa: planejada. Ainda nao ha modelo de mensagem, thread ou chat.
- Memoria: planejada. O diretorio existe, mas nao ha schema ou logica de memoria.
- Ferramentas: planejadas. O diretorio existe, mas nao ha executor ou registry.
- Agentes: planejados. O diretorio existe, mas nao ha runtime de agentes.
- Voz: planejada. O diretorio existe, mas nao ha captura, STT ou TTS.
- IA: planejada. O diretorio existe, mas nao ha provedor integrado.
- Health check: implementado. Verifica backend e conectividade com PostgreSQL.

## Modulos Atuais

Backend:

- `cmd/api`: inicializa configuracao, banco, router HTTP e shutdown gracioso.
- `internal/config`: carrega environment variables e monta endereco HTTP.
- `internal/database`: cria e encapsula pool PostgreSQL.
- `internal/handlers`: registra rotas e implementa `/health`.
- `migrations`: contem schema inicial carregado pelo container Postgres.
- `docker`: contem Dockerfile do backend.

Frontend:

- `app`: App Router do Next.js com layout e pagina inicial.
- `components`: componentes de UI, atualmente `HealthPanel`.
- `services`: chamadas para APIs externas, atualmente health check.
- `types`: tipos TypeScript compartilhados, atualmente `HealthResponse`.
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
5. Backend registra `GET /health`.
6. Frontend inicia e renderiza a pagina inicial.
7. `HealthPanel` chama `getHealth`.
8. `getHealth` consulta `API_INTERNAL_URL` no servidor Next.js e exibe `NEXT_PUBLIC_API_URL` para o usuario.

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
- Health panel.
- Endpoint `/health`.
- Verificacao de banco no health check.

Nao implementado:

- Chat.
- IA.
- Voz.
- Memoria de usuario.
- Ferramentas.
- Agentes.
- Autenticacao.
- Acesso remoto.
