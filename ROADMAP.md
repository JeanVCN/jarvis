# ROADMAP.md

Roadmap vivo do projeto Jarvis.

Este arquivo deve ser atualizado quando funcionalidades forem concluidas, iniciadas ou replanejadas.

## Concluido

- Estrutura inicial do monorepo criada.
- Backend Go com Gin criado.
- Endpoint `GET /health` implementado.
- Health check verifica conexao com PostgreSQL.
- Configuracao do backend via environment variables.
- Encapsulamento inicial de PostgreSQL em `internal/database`.
- Dockerfile do backend com build multi-stage.
- Frontend Next.js com TypeScript e Tailwind criado.
- Tela inicial do frontend exibindo status do backend.
- Service frontend para consultar `/health`.
- Dockerfile do frontend com build multi-stage.
- Docker Compose com `postgres`, `backend` e `frontend`.
- PostgreSQL configurado com volume persistente.
- Migration inicial `001_initial_schema.sql` com tabela `app_metadata`.
- README com instrucoes de execucao.
- Separacao entre URL interna e URL publica da API no frontend.
- Estrutura reservada para `ai`, `voice`, `memory`, `tools`, `agents`, `services` e `repositories`.
- Contrato `LLM` criado em `backend/internal/ai`.
- `OllamaProvider` implementado atras da interface de IA.
- Configuracao de `OLLAMA_URL` e `OLLAMA_MODEL` por environment variables.
- Camada inicial de `services` criada para conversa por texto.
- Endpoint `POST /api/chat` implementado.
- Endpoint `POST /api/chat/stream` implementado com SSE.
- Tela simples de chat no frontend consumindo o endpoint do backend.
- Frontend renderiza resposta do chat em tempo real via stream.
- CORS local configuravel por `CORS_ALLOWED_ORIGIN`.

## Em Andamento

- Fundacao arquitetural do projeto.
- Documentacao viva para continuidade entre sessoes de IA e colaboradores.
- Definicao de fronteiras entre modulos futuros.

## Parcialmente Implementado

- `memory`: diretorio existe, mas nao ha modelo, repositorio ou API de memoria.
- `tools`: diretorio existe, mas nao ha sistema de ferramentas.
- `agents`: diretorio existe, mas nao ha agentes autonomos.
- `ai`: possui interface `LLM` e provider inicial Ollama; ainda nao ha OpenAI ou Gemini.
- `voice`: diretorio existe, mas nao ha captura, transcricao ou sintese de voz.
- `services`: possui caso de uso inicial de chat; ainda nao ha memoria, tools ou agentes.
- `repositories`: diretorio existe, mas ainda nao ha repositorios de dominio.
- `hooks`: diretorio frontend existe, mas ainda nao ha hooks.

## Proximas Etapas

1. Definir modelo de dominio minimo para conversas por texto.
2. Criar migracoes versionadas para entidades reais do Jarvis.
3. Criar camada de repositories para acesso a dados.
4. Evoluir a camada de services para orquestrar conversa, memoria e provedores.
5. Persistir historico de mensagens.
6. Criar UI de chat conversacional com historico local ou persistido.
7. Definir contrato de memoria persistente.
8. Adicionar testes automatizados para backend e frontend.
9. Definir padrao de observabilidade minimo, como logs estruturados e erros padronizados.
10. Adicionar configuracao seletiva de provedores de IA quando OpenAI ou Gemini forem implementados.

## Futuras Evolucoes

- Conversacao por voz.
- Sintese de voz.
- Transcricao de audio.
- Sistema de ferramentas executaveis.
- Agentes autonomos com limites de seguranca.
- Suporte a OpenAI.
- Suporte a Gemini.
- Acesso remoto via Tailscale.
- Autenticacao para acesso remoto.
- Interface responsiva otimizada para celular.
- Painel de configuracoes do assistente.
- Historico de conversas e busca em memoria.

## Dependencias Entre Modulos

- `ai` ja expoe o contrato inicial `LLM`; novos provedores devem implementar essa interface ou uma evolucao compativel dela.
- `memory` depende de schema de banco, repositories e definicao de dados persistidos.
- `agents` depende de `ai`, `tools`, `memory` e regras de permissao.
- `voice` depende de fluxo de conversa definido e, futuramente, de transcricao/sintese.
- UI de chat final depende de historico e modelo de conversa; a tela atual valida a integracao inicial com streaming.
- Acesso remoto depende de autenticacao e configuracao segura de rede.

## Modulos Desenvolviveis em Paralelo

- Frontend de chat pode evoluir em paralelo usando o contrato `POST /api/chat`.
- Repositories e migrations podem evoluir em paralelo com UI, desde que contratos sejam documentados.
- Novos provedores de IA podem evoluir em paralelo atras da interface `LLM`.
- Documentacao e ADRs podem evoluir em paralelo com qualquer modulo.
- Observabilidade basica pode evoluir em paralelo com features de produto.
- Testes de handlers/services podem evoluir em paralelo com a UI.
- Voice deve aguardar pelo menos um fluxo de conversa por texto definido.
- Agents devem aguardar definicao de tools, memory e limites de seguranca.
