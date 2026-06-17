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
- Contrato `Tool` criado em `backend/internal/tools`.
- `ToolRegistry` criado para registrar, listar, buscar e executar ferramentas.
- `TimeTool` implementada para retornar data e hora atual.
- `CalculatorTool` implementada para calculos basicos.
- `ToolService` criado para orquestrar execucao de ferramentas via registry.
- Endpoints `GET /api/tools` e `POST /api/tools/execute` implementados.
- Tela simples de teste de ferramentas criada no frontend.
- `SystemTool` implementada com acoes pre-cadastradas e sem shell livre.
- Chat integrado ao `ToolRegistry` por roteamento local de tools obvias.
- `ChatService` decide entre streaming direto de resposta normal e execucao de tool com resposta deterministica.
- TimeTool, CalculatorTool e SystemTool podem ser acionadas automaticamente pelo chat.
- Streaming de texto preservado para mensagens que nao usam tools.
- Modulo `voice` criado com interfaces `SpeechRecognizer` e `VoiceProvider`.
- Provedores locais de voz por comando configuravel adicionados para STT e TTS.
- Endpoint `POST /api/voice/conversation` implementado para fluxo audio -> texto -> chat/tools -> voz.
- Interface principal redesenhada para experiencia prioritariamente por voz.
- Modo texto mantido recolhido como fallback.
- Interface principal refinada como HUD futurista com nucleo central animado.
- Chat e tools movidos para paineis laterais secundarios.

## Em Andamento

- Fundacao arquitetural do projeto.
- Documentacao viva para continuidade entre sessoes de IA e colaboradores.
- Definicao de fronteiras entre modulos futuros.
- Preparacao para futuras tools como web search, memoria, arquivos e automacao residencial.
- Preparacao para wake word e streaming de audio futuros.

## Parcialmente Implementado

- `memory`: diretorio existe, mas nao ha modelo, repositorio ou API de memoria.
- `tools`: possui contrato, registry, TimeTool, CalculatorTool e SystemTool; ainda nao ha ferramentas externas avancadas.
- `agents`: diretorio existe, mas nao ha agentes autonomos.
- `ai`: possui interface `LLM` e provider inicial Ollama; ainda nao ha OpenAI ou Gemini.
- `voice`: possui contratos, provedores locais por comando, endpoint de conversa e UI de captura; depende de binarios/modelos locais de STT/TTS instalados no ambiente.
- `services`: possui chat com roteamento local para tools obvias e service de tools; ainda nao ha memoria ou agentes.
- `repositories`: diretorio existe, mas ainda nao ha repositorios de dominio.
- `hooks`: diretorio frontend existe, mas ainda nao ha hooks.

## Proximas Etapas

1. Definir modelo de dominio minimo para conversas por texto.
2. Criar migracoes versionadas para entidades reais do Jarvis.
3. Criar camada de repositories para acesso a dados.
4. Persistir historico de mensagens.
5. Criar UI de chat conversacional com historico local ou persistido.
6. Definir contrato de memoria persistente.
7. Adicionar mais testes automatizados para backend e frontend.
8. Definir padrao de observabilidade minimo, como logs estruturados e erros padronizados.
9. Adicionar configuracao seletiva de provedores de IA quando OpenAI ou Gemini forem implementados.
10. Evoluir tool calling por LLM para casos nao obvios sem prejudicar o streaming de mensagens normais.
11. Configurar e documentar modelos locais recomendados para Faster Whisper e Piper.
12. Adicionar wake word local.
13. Evoluir voz para streaming bidirecional de audio.

## Futuras Evolucoes

- Wake word para conversacao por voz.
- Sintese de voz.
- Transcricao de audio.
- Sistema de ferramentas executaveis com mais integracoes externas.
- Ferramentas futuras como WebSearchTool, HomeAssistantTool, MemoryTool e FileTool.
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
- `tools` expoe um `ToolRegistry` que deve ser usado por services, IA e agentes para executar ferramentas sem depender de implementacoes concretas.
- `agents` depende de `ai`, `tools`, `memory` e regras de permissao.
- `voice` depende de binarios/modelos locais configurados por env vars e usa o fluxo de chat existente.
- UI de chat final depende de historico e modelo de conversa; a tela atual valida a integracao inicial com streaming.
- Acesso remoto depende de autenticacao e configuracao segura de rede.

## Modulos Desenvolviveis em Paralelo

- Frontend de chat pode evoluir em paralelo usando o contrato `POST /api/chat`.
- Repositories e migrations podem evoluir em paralelo com UI, desde que contratos sejam documentados.
- Novos provedores de IA podem evoluir em paralelo atras da interface `LLM`.
- Documentacao e ADRs podem evoluir em paralelo com qualquer modulo.
- Observabilidade basica pode evoluir em paralelo com features de produto.
- Testes de handlers/services podem evoluir em paralelo com a UI.
- Voice pode evoluir em paralelo com UI e providers, desde que preserve os contratos `SpeechRecognizer` e `VoiceProvider`.
- Agents devem aguardar definicao de tools, memory e limites de seguranca.
