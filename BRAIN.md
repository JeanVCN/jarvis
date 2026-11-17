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

- Monorepo com `backend/`, `frontend/` e `voice-agent/`.
- Backend organizado em `cmd/` e `internal/`.
- `cmd/api` contem o ponto de entrada HTTP.
- Configuracao do backend centralizada em `internal/config`.
- Conexao PostgreSQL encapsulada em `internal/database`.
- Rotas HTTP ficam em `internal/handlers`.
- Middlewares HTTP compartilhados ficam em `backend/internal/middleware`.
- `backend/internal/ai` contem o contrato `LLM` e a implementacao inicial `OllamaProvider`.
- `backend/internal/services` contem o caso de uso inicial de chat por texto, tool calling simples e service de tools.
- `backend/internal/tools` contem o contrato `Tool`, o `ToolRegistry` e ferramentas iniciais, incluindo `SystemTool`.
- `backend/internal/voice` contem os contratos `SpeechRecognizer` e `VoiceProvider`, provider local por comando configuravel e provider STT HTTP para servidor local persistente.
- Diretorios `voice`, `memory`, `tools`, `agents` e `repositories` seguem como limites arquiteturais para evolucao futura.
- Frontend usa App Router do Next.js em `frontend/app`.
- Frontend separa componentes, services, hooks e types.
- O frontend consulta o backend pelo service `frontend/services/health.ts`.
- O frontend possui service de chat para chamar `POST /api/chat`.
- O frontend usa `POST /api/chat/stream` para renderizar respostas do chat em tempo real via SSE.
- O frontend possui service de tools para chamar `GET /api/tools` e `POST /api/tools/execute`.
- O frontend possui interface principal de voz com estados `Idle`, `Listening`, `Thinking` e `Speaking`.
- A interface de voz usa VAD local no navegador para detectar fala/silencio e pode operar em standby aguardando wake word.
- A home do frontend deve priorizar um HUD de voz com nucleo central animado; chat e tools devem permanecer secundarios.
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
- Tools devem ficar atras da interface `tools.Tool` e ser executadas pelo `ToolRegistry`.
- A integracao entre LLM e tools ocorre no `ChatService`, sem acoplar Ollama diretamente as ferramentas.
- Tool calling atual usa roteamento local para casos obvios; planejamento por JSON pelo LLM fica como evolucao futura para casos nao obvios.
- `SystemTool` deve executar apenas acoes pre-cadastradas e nao deve permitir comandos arbitrarios.
- Providers de voz devem ficar atras de `SpeechRecognizer` e `VoiceProvider`.
- STT/TTS locais sao configurados por env vars e nao devem acoplar handlers a Faster Whisper, Sherpa-ONNX, Piper ou outro provider especifico.
- Audio bruto nao deve ser persistido nem logado por padrao; providers locais devem processar em memoria ou arquivos temporarios descartaveis.
- Streaming de chat usa SSE como caminho inicial; WebSocket fica reservado para fluxos bidirecionais futuros.
- Voz realtime deve ficar isolada em `voice-agent`, com Pipecat/LiveKit como direcao preferida para conversa natural.
- O `voice-agent` deve chamar `POST /api/chat/stream` no backend Go; tools, permissoes, memoria e comandos continuam sob autoridade do Jarvis Core.
- O fluxo batch `POST /api/voice/conversation` permanece como fallback/debug enquanto a experiencia principal migra para transporte realtime.

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

## Direção definida em 2026-11-13

- Uso pessoal entre dispositivos; servidor no notebook Linux i5-1135G7, 8 GB,
  Iris Xe; Go/PostgreSQL/Next.js ativos em conjunto.
- Orçamento adicional máximo R$ 0; sem fallback pago ou cobrança automática.
- Acesso privado pela tailnet. Tailscale Serve cobre HTTPS, não o transporte ICE de mídia.
- A primeira entrega nova é o laboratório web `/lab`; Android/Linux nativos são posteriores.
- Modelos finais serão escolhidos por testes no notebook. Estado e limitações em
  `docs/voice-lab/README.md`; não confundir perfis sintéticos com inferência real.

## Direção definida em 2026-11-15 — Primeira utilidade residencial

Status atual: **pausada até o usuário ter Android**. Preservar o probe e os
detalhes abaixo como checkpoint; a prioridade ativa foi substituída pela
integração Jarvis↔OpenCode registrada ao final deste documento e no `ROADMAP.md`.

- Começar por ligar/desligar uma única lâmpada no Google Home via NovaDigital
  Smart. Home Assistant não é infraestrutura existente confirmada.
- Avaliar app Android com Home APIs oficiais; o SDK beta foi obtido e o probe
  somente leitura compilou via Gradle Wrapper/JDK 17. OAuth e teste físico
  continuavam pendentes no checkpoint; ver `docs/home-apis/README.md`.
- Android passa a ter papel de ponte/diagnóstico antes do cliente nativo de voz.
  Rota condicionada à descoberta real da lâmpada; não presumir compatibilidade.
- Core permanece autoridade de ferramentas/permissões/journal. Consentimento
  Google não substitui autorização por dispositivo/ação no Jarvis.
- Não alterar pareamento nem executar testes físicos sem autorização; preservar
  credenciais e identificadores domésticos. Não repetir ações após reconexão.
- TV/TV Box vêm depois; APIs do Google não garantem controle de qualquer app
  ou botão. Fases, critérios e limites em `ROADMAP.md` e `ARCHITECTURE.md`.

## Direção definida em 2026-11-15 — Programar pelo celular via Jarvis

- Jarvis será a interface/ponto de controle; OpenCode V2 será o executor inicial
  para tarefas de código no computador que contém os projetos. Não criar agora
  um harness completo próprio nem assumir controle universal da GUI.
- O usuário deve poder escolher projeto, iniciar/continuar sessão, acompanhar
  estado e resposta/diff, interromper e revisar pedidos de aprovação pelo Jarvis.
  A primeira implementação cobre sessão, prompt, consulta/reconciliação básica e
  decisões `once`/`reject`; eventos ao vivo e interrupção ainda faltam.
- O Jarvis não terá login. O acesso remoto será pela tailnet com regras por
  dispositivo; configuração e teste de acesso permitido/negado ainda pendentes.
  O frontend valida origem de mutações, mas Core/bridge/OpenCode precisam ficar
  inacessíveis diretamente ao cliente remoto.
- `opencode-bridge` usa `@opencode/client` 2.2.7, bind loopback, token interno e
  allowlist. Criação de sessão/prompt e permissões requerem
  `JARVIS_OPENCODE_WRITE_ENABLED=true` no Core e bridge; permanece false por
  padrão. Operações ficam registradas no PostgreSQL antes da chamada externa.
  Build/testes com mocks passaram; serviço real, contrato de idempotência e
  topologia final ainda precisam de validação em projeto descartável.
- Eventos OpenCode são live-only; buscar estado/mensagens/diff após reconexão e
  reconciliar timeout antes de reenviar. Começar com aprovações pontuais; shell
  executa com privilégios do usuário local e permissões não equivalem a sandbox.
- Na API V2 2.2.7, `session.prompt` retorna item inbox, não resposta final; as
  permissões incluem a decisão persistente `always`. Não repetir prompts em
  timeout nem expor `always` na primeira UI. Contrato/testes:
  `docs/opencode-integration.md`.
- Próxima etapa bloqueante: configurar acesso Tailscale por dispositivo e
  qualificar permissões, prompt e reconciliação no OpenCode real antes de ativar
  escrita. Plano em `ROADMAP.md` C0–C4; arquitetura em
  `ARCHITECTURE.md`; decisão em `DECISIONS.md`.
- Avaliação 2026-11-16: a fatia de escrita já existe atrás de flag e passou testes
  com mocks, mas não foi usada com OpenCode real nem telefone. Não confundir
  proximidade de implementação com prontidão remota. Retomar IoT após o gate piloto
  do `ROADMAP.md` e quando houver Android; antes disso somente planejamento pode
  avançar na trilha residencial.
