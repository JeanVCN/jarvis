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

### 2026-06-17 - ToolRegistry Como Fronteira De Ferramentas

Status: aceita.

Contexto:

- A fase 3 precisa permitir que o Jarvis execute ferramentas registradas.
- A proxima fase deve permitir que o LLM escolha ferramentas sem depender de implementacoes concretas.
- O projeto ja usa contratos internos para evitar acoplamento direto entre handlers, services e provedores externos.

Decisao:

- Criar `backend/internal/tools` com interface `Tool`.
- Usar `ToolRegistry` como ponto unico para registrar, listar, buscar e executar ferramentas.
- Criar `ToolService` para orquestrar uso de ferramentas pela camada HTTP e por futuros fluxos de IA.
- Registrar `TimeTool` e `CalculatorTool` na inicializacao do backend.
- Nao conectar automaticamente Ollama e tools nesta fase.

Impacto:

- Novas tools podem ser adicionadas implementando `Tool` e registrando a instancia no bootstrap.
- Handlers e services nao dependem de tools concretas.
- A futura orquestracao LLM -> ToolRegistry pode ser implementada sem refatorar handlers ou ferramentas existentes.

### 2026-06-17 - Tool Calling Simples Por JSON

Status: substituida parcialmente por roteamento local para tools obvias.

Contexto:

- O Jarvis precisa decidir entre responder normalmente e executar uma tool.
- A solucao deve funcionar com Ollama e com a interface `ai.LLM` atual.
- O projeto nao deve adotar frameworks externos de agentes nesta fase.

Decisao:

- Implementar a orquestracao no `ChatService`.
- Instruir o LLM por prompt a responder texto normal ou JSON no formato `{"tool":"nome","input":"entrada"}`.
- Quando houver JSON valido, executar a tool via `ToolService` e `ToolRegistry`.
- Fazer uma segunda chamada ao LLM com o resultado da tool para gerar resposta final amigavel.
- Manter o streaming SSE do frontend, mas enviar a resposta final como um unico delta nesta fase.

Impacto:

- TimeTool, CalculatorTool e SystemTool podem ser usadas automaticamente pelo chat.
- O `ToolRegistry` continua sendo a unica fonte de registro e execucao de ferramentas.
- Fluxos com multiplas chamadas de tools podem ser adicionados depois sem trocar handlers.

### 2026-06-17 - Roteamento Local Para Tools Obvias

Status: aceita.

Contexto:

- A decisao por JSON feita pelo LLM antes de toda resposta adicionava uma chamada extra ao Ollama.
- Essa chamada extra aumentava a latencia tambem para mensagens comuns.
- O endpoint de streaming acabava enviando a resposta apenas depois do ciclo completo, perdendo o beneficio do SSE.
- As tools atuais cobrem intencoes simples e reconheciveis: hora/data, calculos basicos e acoes controladas do sistema.

Decisao:

- `ChatService` deve detectar localmente apenas casos obvios para `TimeTool`, `CalculatorTool` e `SystemTool`.
- Mensagens comuns devem chamar `LLM.Stream` diretamente no endpoint de stream.
- Quando uma tool for usada, a execucao continua passando por `ToolService` e `ToolRegistry`.
- Tools simples devem retornar respostas deterministicas curtas, sem chamada extra ao LLM.

Impacto:

- Mensagens normais voltam a ter streaming real e menor latencia percebida.
- Comandos de sistema deixam de depender do julgamento do LLM depois que a tool ja foi escolhida.
- Tools atuais continuam automaticas sem acoplar o sistema a implementacoes concretas.
- Um planejador futuro por LLM pode ser adicionado para casos nao obvios, desde que preserve o caminho rapido de streaming.

### 2026-06-17 - SystemTool Com Acoes Pre-Cadastradas

Status: aceita.

Contexto:

- O Jarvis deve poder executar acoes no sistema operacional.
- Executar comandos arbitrarios criaria risco alto.
- A entrada do usuario ou do LLM nao deve virar comando de shell.

Decisao:

- Criar `SystemTool` com acoes explicitamente registradas.
- A entrada da tool representa apenas o nome da acao.
- Usar comandos e argumentos definidos no codigo da action, sem shell livre.
- Acoes iniciais: `open_browser`, `open_vscode`, `open_terminal`, `open_spotify` e `lock_screen`.

Impacto:

- A tool pode abrir programas e bloquear a tela dentro de uma lista controlada.
- Acoes futuras podem ser adicionadas de forma explicita e revisavel.
- Inputs desconhecidos retornam erro e nao sao executados.

### 2026-06-17 - Voz Local Por Contratos E Comandos Configuraveis

Status: aceita.

Contexto:

- A experiencia principal do Jarvis deve passar a ser voz.
- STT e TTS devem funcionar localmente, sem dependencia obrigatoria de APIs pagas.
- Faster Whisper e Piper sao boas opcoes iniciais, mas o projeto deve permitir troca futura.

Decisao:

- Criar `backend/internal/voice` com os contratos `SpeechRecognizer` e `VoiceProvider`.
- Implementar providers por comandos locais configuraveis por env vars.
- Usar placeholders como `{input}`, `{output}`, `{language}`, `{model}` e `{voice}` para adaptar CLIs diferentes.
- Executar os comandos diretamente com `exec.CommandContext`, sem shell.
- Criar `VoiceService` para orquestrar STT -> ChatService -> TTS.
- Criar `POST /api/voice/conversation` como endpoint inicial de conversa por audio.

Impacto:

- Faster Whisper, Piper ou alternativas locais podem ser conectados sem alterar handlers.
- Ambientes sem binarios/modelos configurados retornam `voice_provider_unavailable`.
- A base fica preparada para wake word e streaming de audio em fases futuras.

### 2026-06-17 - STT Local Otimizado Para Baixa Latencia

Status: substituida parcialmente por servidor STT local persistente.

Contexto:

- O STT inicial executa um comando local por fala.
- Cada execucao inicia Python e carrega o modelo Whisper novamente.
- O modelo `small` melhora precisao, mas aumenta a latencia em comandos curtos.

Decisao:

- Usar `tiny` como modelo STT default em desenvolvimento.
- Adicionar um wrapper local rapido com `beam_size=1`, `best_of=1` e `condition_on_previous_text=false`.
- Manter `base` e `small` como opcoes configuraveis por `VOICE_STT_MODEL` quando a precisao for mais importante.

Impacto:

- A transcricao de comandos curtos deve responder mais rapido.
- A precisao pode cair em frases longas ou audio ruidoso.
- A evolucao recomendada continua sendo um provider STT persistente para manter o modelo carregado entre requisicoes.

### 2026-06-18 - Voz Privada Com VAD Local E STT Persistente

Status: aceita.

Contexto:

- O STT por comando ainda carregava o modelo a cada fala.
- O modelo `tiny` reduzia latencia, mas degradava muito a precisao em portugues.
- A experiencia desejada exige detectar fim de fala sem clique e evitar execucao acidental de comandos.
- O projeto deve preservar privacidade, sem enviar audio para APIs externas.

Decisao:

- Adicionar um provider `HTTPSpeechRecognizer` para falar com servidor STT local persistente.
- Criar `scripts/jarvis-stt-server`, que carrega Faster Whisper uma vez, escuta apenas em `127.0.0.1` e descarta arquivos temporarios apos transcricao.
- Usar `small` como modelo STT default no perfil F5 para recuperar precisao.
- Implementar VAD local no frontend com Web Audio API para detectar inicio/fim de fala.
- Implementar wake word por transcricao com `wakeRequired=true`; sem wake word, o backend retorna `ignored=true` sem chamar LLM, tools ou TTS.

Impacto:

- O custo de carregar o modelo sai do caminho de cada fala e passa para a subida do servidor STT.
- A captura deixa de exigir clique para finalizar a fala.
- O audio continua local e nao e persistido por padrao.
- A wake word atual ainda depende da transcricao; keyword spotting acustico dedicado com Sherpa-ONNX permanece como evolucao recomendada.

### 2026-06-17 - Interface Principal Orientada A Voz

Status: aceita.

Contexto:

- A tela principal anterior era centrada em chat e ferramentas manuais.
- A nova fase pede que voz seja a experiencia principal e texto seja fallback.

Decisao:

- Criar `VoiceAssistant` como componente principal da home.
- Usar estados visuais `Idle`, `Listening`, `Thinking` e `Speaking`.
- Animar `Listening` com volume do microfone e `Speaking` com volume do audio reproduzido.
- Manter `ChatPanel` recolhido sob `Modo Texto`.
- Manter `ToolsPanel` recolhido para desenvolvimento e debug.

Impacto:

- A primeira tela passa a parecer um assistente virtual, nao um formulario administrativo.
- O modo texto continua disponivel para fallback e depuracao.
- A UI fica pronta para wake word e streaming de audio futuros.

### 2026-06-17 - Home Como HUD De Assistente

Status: aceita.

Contexto:

- A primeira versao da interface de voz ainda parecia um formulario ou painel administrativo.
- A experiencia principal deve transmitir assistente pessoal, IA avancada e sistema futurista.

Decisao:

- Remover texto hero e paineis centrais da primeira tela.
- Usar um nucleo central animado como elemento principal.
- Exibir hora/data no canto superior esquerdo e estado no canto superior direito.
- Colocar o botao circular de voz no centro inferior.
- Mover modo texto, tools e health para paineis laterais secundarios.
- Usar paleta escura com azul/ciano e evitar verde forte na tela principal.

Impacto:

- Ao abrir a aplicacao, a primeira impressao passa a ser de assistente de IA, nao de dashboard.
- O modo texto continua disponivel sem competir com a experiencia de voz.

### 2026-06-28 - Voz Realtime Em Modulo Separado

Status: aceita.

Contexto:

- O fluxo batch de voz melhorou, mas ainda apresentou latencia alta, baixa confiabilidade de reconhecimento e transcricao intermitente.
- Assistentes de voz usaveis normalmente usam audio continuo, turn detection, STT streaming, LLM streaming, TTS streaming e capacidade de interrupcao.
- O Jarvis Core ja expoe `POST /api/chat/stream`, que preserva tools e respostas deterministicas por SSE.

Decisao:

- Criar `voice-agent/` como servico Python separado para a camada realtime.
- Manter o backend Go como Jarvis Core e autoridade para tools, memoria futura, permissoes e comandos locais.
- Fazer o agente chamar `POST /api/chat/stream` em vez de executar comandos diretamente.
- Emitir tokens de sessao LiveKit no `voice-agent` para clientes desktop/Android entrarem em salas WebRTC sem expor o segredo para o frontend.
- Usar Pipecat/LiveKit como direcao preferida para transporte/pipeline realtime.
- Manter `POST /api/voice/conversation` como fallback batch e caminho de debug.

Impacto:

- A experiencia principal de voz deixa de depender do upload de trecho completo antes de iniciar resposta.
- Desktop e Android podem compartilhar o mesmo servico local via transporte realtime.
- A migracao fica reversivel e incremental, sem descartar imediatamente o trabalho de STT/TTS atual.

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

### 2026-11-13 — Comparação reproduzível de voz antes de selecionar modelos

Status: aceita para a infraestrutura; seleção de modelos pendente de testes no notebook.

Contexto:

- Pipeline batch e IA local não entregaram experiência utilizável.
- Máquina alvo i5-1135G7/8 GB/Linux, uso individual e orçamento adicional zero.
- É necessário separar qualidade do modelo de latência de transporte e processamento.

Decisão:

- Preservar fluxos legados e adicionar `/lab`, SmallWebRTC e contratos `/api/v2`.
- Manter Go/PostgreSQL como autoridade de histórico, contexto e ferramentas.
- Comparar Whisper base/small INT8, Kokoro e Pocket pelo mesmo caminho de áudio.
- Usar Silero com silêncio fixo ou Smart Turn v3.2, carregados localmente.
- Oferecer scripted, Groq Free, Ollama e adaptador experimental Codex App Server;
  exigir ativação/credenciais e inferência real antes de declarar provedor elegível.
- Nunca ativar fallback pago; não reutilizar credenciais da tarefa Codex.
- Bloquear ferramentas de sistema por padrão; não repetir ferramentas após reconexão.
- Persistir eventos/turnos e distinguir texto gerado de reprodução estimada.

Impacto:

- Migrações são aplicadas também em bancos existentes, com lock e registro de versões.
- O histórico novo persiste; os endpoints antigos continuam disponíveis.
- Variantes finais, sotaque PT-BR, licença individual Pocket, cotas e métricas
  dependem das verificações documentadas, sem promessas baseadas no executor remoto.
- Preservam-se verificação TLS e políticas de rede/navegador do ambiente.
