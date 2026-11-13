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
- Servidor STT local persistente adicionado para manter modelo carregado entre falas.
- VAD local no frontend adicionado para detectar fim de fala sem clique.
- Wake word simples por transcricao adicionada para ignorar audio sem chamada ao Jarvis.
- Modulo `voice-agent` criado como fronteira separada para evoluir voz realtime com Pipecat/LiveKit.
- `voice-agent` ja possui cliente do Jarvis Core via `POST /api/chat/stream`, modo console, servidor local de health/smoke test e testes do parser SSE.
- `voice-agent` emite tokens de sessao LiveKit para clientes desktop/Android via `POST /livekit/session`.
- Frontend chama o `voice-agent` por `NEXT_PUBLIC_VOICE_AGENT_URL`, entra na sala LiveKit e publica o microfone a partir do HUD principal.
- `voice-agent` possui participante realtime que entra na sala LiveKit, assina faixas de audio e contabiliza frames recebidos.
- `voice-agent` segmenta audio realtime por VAD simples, envia turnos para STT local e chama o Jarvis Core quando a wake word e detectada.

## Em Andamento

- Fundacao arquitetural do projeto.
- Documentacao viva para continuidade entre sessoes de IA e colaboradores.
- Definicao de fronteiras entre modulos futuros.
- Preparacao para futuras tools como web search, memoria, arquivos e automacao residencial.
- Preparacao para keyword spotting acustico e streaming de audio futuros.
- Migracao da experiencia principal de voz do fluxo batch para um agente realtime.

## Parcialmente Implementado

- `memory`: diretorio existe, mas nao ha modelo, repositorio ou API de memoria.
- `tools`: possui contrato, registry, TimeTool, CalculatorTool e SystemTool; ainda nao ha ferramentas externas avancadas.
- `agents`: diretorio existe, mas nao ha agentes autonomos.
- `ai`: possui interface `LLM` e provider inicial Ollama; ainda nao ha OpenAI ou Gemini.
- `voice`: possui contratos, provedores locais por comando/HTTP, servidor STT local, endpoint de conversa, VAD no frontend e UI de captura; depende de binarios/modelos locais de STT/TTS instalados no ambiente e fica como fallback batch.
- `voice-agent`: possui base separada para voz realtime, emissao de token LiveKit, integracao inicial com o HUD e participante que assina audio; ja transcreve turnos via STT HTTP local, mas ainda nao sintetiza audio de resposta de volta para a sala.
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
11. Validar manualmente transporte realtime LiveKit + STT + Jarvis Core em ambiente local com rede Docker funcional.
12. Conectar TTS/audio de resposta do `voice-agent` de volta para a sala LiveKit e medir latencia por etapa.
13. Configurar e documentar modelos locais recomendados para Faster Whisper, Sherpa-ONNX, Piper/Kokoro e alternativas.
14. Evoluir wake word por transcricao para keyword spotting acustico local.
15. Criar cliente Android/APK apontando para o servico local via transporte realtime.

## Futuras Evolucoes

- Keyword spotting acustico para wake word.
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
- `voice` batch depende de binarios/modelos locais configurados por env vars e usa o fluxo de chat existente.
- `voice-agent` depende do Jarvis Core via `POST /api/chat/stream`; transporte realtime deve permanecer desacoplado de tools e comandos diretos.
- UI de chat final depende de historico e modelo de conversa; a tela atual valida a integracao inicial com streaming.
- Acesso remoto depende de autenticacao e configuracao segura de rede.

## Modulos Desenvolviveis em Paralelo

- Frontend de chat pode evoluir em paralelo usando o contrato `POST /api/chat`.
- Repositories e migrations podem evoluir em paralelo com UI, desde que contratos sejam documentados.
- Novos provedores de IA podem evoluir em paralelo atras da interface `LLM`.
- Documentacao e ADRs podem evoluir em paralelo com qualquer modulo.
- Observabilidade basica pode evoluir em paralelo com features de produto.
- Testes de handlers/services podem evoluir em paralelo com a UI.
- Voice batch pode evoluir em paralelo com UI e providers, desde que preserve os contratos `SpeechRecognizer` e `VoiceProvider`.
- Voice realtime pode evoluir em paralelo no `voice-agent`, desde que o backend Go continue sendo a autoridade para tools, memoria e permissoes.
- Agents devem aguardar definicao de tools, memory e limites de seguranca.

## Atualização 2026-11-13 — Laboratório de voz contínua

Implementado:

- `/lab` com seleção de STT/TTS/voz/turn detection/provedor, histórico e exportação de eventos.
- Conversas, turnos, eventos SSE retomáveis e journal de ferramentas no PostgreSQL.
- Migrações aditivas, recuperação de turnos interrompidos e trava de instância do Core.
- SmallWebRTC, Silero, Smart Turn v3.2, áudio progressivo e descarte por geração.
- Adaptadores CPU Whisper base/small, Kokoro PT-BR e Pocket português.
- Provedores scripted, Groq, Ollama e Codex App Server experimental.
- Scripts de instalação/execução, corpus, WER, TTS isolado, recursos e smoke tests.

Pendente de evidência externa (não considerar concluído):

- Downloads e inferência dos pesos STT/TTS bloqueados pela rede do executor.
- Licença individual da voz Pocket, enquadramento OAuth e elegibilidade/cota da conta.
- Inferência real de ChatGPT/Groq/Ollama e comparação dos modelos reais elegíveis.
- Áudio no browser físico; Chromium gerenciado bloqueia UDP WebRTC neste executor.
- Notebook de 8 GB, qualidade PT-BR, metas de latência e interrupção acústica.
- ICE/Tailscale/Android em Wi-Fi e rede móvel; seleção final dos perfis.

Candidatos de segunda rodada permanecem condicionais, conforme a matriz em
`docs/voice-lab/README.md`. LiveKit e voz batch não foram removidos.
