# ARCHITECTURE.md

Documentacao arquitetural do estado atual do Jarvis.

## Estado de referência — 2026-11-16

O fluxo atual de laboratório usa Next.js `/lab`, Python SmallWebRTC para áudio
e Go `/api/v2` para conversa, provedores e ferramentas, com histórico, turnos,
eventos e journal no PostgreSQL. Os fluxos batch e LiveKit abaixo são legados/
alternativos preservados, não a única arquitetura atual. As descrições antigas
de ausência de histórico/migrações incrementais não se aplicam a `/api/v2`.
GPT-6-Luna via Codex teve resposta e calculadora reais validadas em 2026-11-14;
isso não comprova qualidade/latência da voz em aparelho físico.

### Extensão prioritária: delegação de tarefas de código ao OpenCode

**Primeira implementação concluída; qualificação real pendente.** `opencode-bridge`,
Core e `/code` implementam leitura e escrita guardada por flag. Fases C0–C4 em
`ROADMAP.md`.

```text
Celular/desktop → UI Jarvis (Tailscale/rede privada) → Core Go (loopback)
                                  ↓ bridge bearer interno, loopback
                           adaptador TypeScript
                                  ↓ cliente OpenCode V2
                           serviço OpenCode local → projetos/terminal
```

- Jarvis é a interface, orquestrador, registro da delegação e
  autoridade para escolher projetos permitidos. OpenCode executa tarefas de
  programação no computador que contém os repositórios. Não é assumido como
  controlador genérico da interface gráfica do desktop.
- Direção escolhida: integrar OpenCode V2 em vez de criar agora um harness
  completo próprio. Implementado: adaptador local TypeScript usando
  `@opencode/client`, iniciado opcionalmente pelo script host. Criar sessão,
  enviar prompt, acompanhar por consulta, ler resposta/diff e responder `once`/
  `reject` estão atrás de `JARVIS_OPENCODE_WRITE_ENABLED`, desligado por padrão.
  SDK embutido não foi escolhido; Compose e topologia/proxy ainda não foram
  validados. Core Go não deve executar shell nem receber credenciais de provedores.
- Contrato atual do adaptador: listar projetos/sessões, criar/continuar sessão,
  ler status/mensagens/diff e intermediar aprovações pontuais. Interrupção e
  eventos ao vivo permanecem pendentes. Diretórios vêm da allowlist no bridge,
  nunca de paths livres fornecidos pelo modelo/browser.
- O app não terá login Jarvis. O navegador acessa pela máquina local ou rede
  privada, com Tailscale configurado para permitir somente dispositivos
  escolhidos. Essa regra ainda precisa de teste prático. O frontend valida origem
  em mutações, mas Core não autentica usuário; mantenha frontend como única entrada
  remota e Core/rewrite/bridge/OpenCode em loopback.
- A primeira implementação tem testes de contrato com mocks, não execução real.
  Antes de ativar escrita, validar semântica de permissões, ID/correlação do
  prompt, reconciliação e reinício contra OpenCode 2.2.7 em projeto descartável.
  Não exponha diretamente a porta do Core, inclusive quando as rotas OpenCode
  estiverem habilitadas. O bearer entre Core e bridge é autenticação de serviço,
  não autenticação do usuário.
- Eventos OpenCode via cliente são ao vivo, sem replay/reconexão automática. Usar
  eventos para atualização rápida e consultar estado/mensagens/diff persistidos
  pela API após reconexão. Reconciliar timeout antes de repetir prompt.
- Mostrar estado final, resposta e diff só após confirmação pela sessão OpenCode.
  Aprovações começam pontuais; cancelamento não desfaz ações já realizadas.
- Permissões do agente não isolam o shell: processos executam com autoridade do
  usuário do computador. O serviço OpenCode não deve ser exposto à internet nem
  ficar acessível ao telefone fora da fronteira prevista.
- Referências V2: [SDK](https://opencode.ai/docs/build/sdk/),
  [client](https://opencode.ai/docs/build/client/), [API](https://opencode.ai/docs/api/),
  [Web/serviço](https://opencode.ai/docs/cli/web/),
  [permissões](https://opencode.ai/docs/permissions/),
  [políticas](https://opencode.ai/docs/policies/).
- Bridge usa `@opencode/client` 2.2.7 e `Service.discover`, faz bind somente em
  `127.0.0.1`, exige bearer interno e resolve projetos por allowlist. Além das
  rotas read-only, cria sessões, envia prompts, lê respostas/diff limitados e
  intermedeia permissões pontuais. Toda escrita está atrás de
  `JARVIS_OPENCODE_WRITE_ENABLED`, false por padrão; exige habilitação tanto no
  Core quanto no bridge. A persistência/journal fica no PostgreSQL.
- `/code` permite selecionar projeto/sessão, continuar tarefa, acompanhar por
  polling e reconciliar estados incertos. Stream de eventos e interrupção explícita
  ainda não existem. Compose não habilita o bridge; a topologia atual é o script
  host. Detalhes de configuração em `opencode-bridge/README.md`.
- O app não terá login Jarvis. O frontend verifica origem nas mutações; Core não
  autentica usuário. O controle remoto depende de Tailscale Serve e grants/ACLs
  restritos, ainda não testados. Core, bridge e OpenCode não podem ficar acessíveis
  diretamente ao cliente.
- `session.prompt` confirma entrada na inbox, não conclusão. Embora a operação
  tenha ID Jarvis persistido e seja reconciliada, sua idempotência no servidor
  OpenCode ainda não foi comprovada. Não repetir automaticamente chamadas incertas.
  Ver pendências em `docs/opencode-integration.md`.

#### Prontidão e mudança de prioridade — 2026-11-16

O código está perto de uma prova piloto, mas o uso remoto ainda não foi validado:
faltam OpenCode real, grants Tailscale por dispositivo e um ciclo completo pelo
telefone. O gate e a recomendação para retomar IoT estão em `ROADMAP.md`. Após o
gate, faz sentido devolver prioridade à IoT quando houver Android; sem Android,
OAuth, discovery real da lâmpada e validação física continuam bloqueados.

### Extensão residencial pausada: app Android e Google Home

**Pausada até o usuário ter Android disponível.** Checkpoint e critérios anteriores
preservados em `ROADMAP.md` (A0–A5); não iniciar OAuth, discovery ou atuação agora.

```text
Texto/voz → Jarvis Core (Go) → ToolRegistry + política + journal
                                      ↕ canal privado autenticado
                                 App Android
                                      ↕ consentimento Google
                                 Google Home APIs → dispositivo
```

- O app Android fica em `android/home-probe`, MVP de diagnóstico e não reescrita
  do frontend/agente de voz. Google lista Android Studio 2024.2.1+ e ADB como
  requisitos oficiais; o Gradle Wrapper CLI compilou o probe com JDK 17 após
  instalar o SDK beta em Maven local. O probe registra tipos de luz, lê nome/
  cômodo e não contém comandos. OAuth e validação física pendentes. Detalhes em
  `docs/home-apis/README.md`.
- O Core permanece autoridade de permissões e journal. A implementação atual
  restringe `/api/v2` a `time` e `calculator`; automação exige evolução explícita
  dessa política e das ferramentas expostas aos provedores, não apenas registro.
- O Android solicita consentimento pelas Home APIs e mantém a sessão Google
  no aparelho; o Core não recebe tokens Google. Autorização da casa pelo Google
  não autoriza todos os dispositivos no Jarvis: há allowlist adicional, limitada
  inicialmente a uma lâmpada e comandos explícitos on/off.
- Descoberta produz um catálogo de identificadores opacos, casas/cômodos,
  nomes e capacidades. O usuário confirma o vínculo com um apelido do Jarvis.
  Nomes não identificam univocamente nem concedem permissão. Mudança de conta,
  casa, identificador ou remoção exige revalidação, não associação por nome.
- A comunicação terá pareamento autenticado, revogação, validade de comandos,
  correlação/deduplicação e estado de disponibilidade. Rede local/tailnet não
  substitui autenticação de aplicação. Sem exposição pública ou segredos em logs.
- Transporte, persistência do catálogo, esquema de mensagens e mecanismo de
  segundo plano ainda precisam ser definidos/testados em A2. Preferir conexão
  iniciada pelo app; não assumir que Android permite um daemon permanente.
- O contrato deverá distinguir comando enviado/aceito de estado observado com
  instante de leitura, além de falha e resultado incerto. Sem retransmissão cega
  após timeout ou reconexão, sem promessa de atuação exatamente uma vez.
- O journal atual não é, sozinho, garantia de deduplicação entre Core, ponte e
  dispositivo. Se necessário, adicionar migrações/estados sem descartar histórico.
- Ausência do Android, rede, consentimento ou serviço Google deixa essa rota
  indisponível; ela não é controle local/offline garantido. Revogar acesso deve
  impedir novos comandos. Cancelamento de áudio não reverte atuação física.

### Limites e porta de viabilidade

O dispositivo conhecido é “Luz Do Quarto”, vinculado no Google Home pelo serviço
NovaDigital Smart (nome de origem “luzinha”). Não há Home Assistant instalado
confirmado, compatibilidade Tuya comprovada ou API NovaDigital validada. Não é
necessário coletar modelo/MAC/ID virtual para tentar descoberta pelas Home APIs.

A1 deve comprovar leitura e comandos disponíveis sem atuar. Se a lâmpada não
for exposta/controlável, suspender a ponte completa e discutir outra integração,
sem reset/desvinculação automática. Tipos e traits na documentação não garantem
compatibilidade de um aparelho concreto. TV/TV Box e automação de interface de
apps ficam para análise posterior, não são capacidades presumidas desta ponte.

Referências oficiais a conferir em A0: [início Android](https://developers.home.google.com/apis/android/get-started),
[permissões](https://developers.home.google.com/apis/android/permissions),
[app de exemplo](https://developers.home.google.com/apis/android/sample-app/build),
[tipos suportados](https://developers.home.google.com/apis/android/supported-device-types)
e [comandos](https://developers.home.google.com/apis/android/device/control).

## Canal pessoal via WhatsApp

`backend/whatsapp` é um processo Go independente. O dispositivo vinculado
via `whatsmeow` recebe eventos por uma conexão de saída. A entrada aceita
apenas o chat **Você** da conta pareada cujo número coincide com a
configuração. O processo envia as respostas pela mesma conexão, com marca
`Jarvis:` para evitar respostas em loop.

| Entrada | Destino |
|---|---|
| `/code` digitado | Alternância local e persistida. O comando não aciona inferência. |
| Texto ou transcrição PTT no chat normal | Thread Codex persistente via `internal/conversation`; o SQLite guarda somente seu ID e o conteúdo fica no `CODEX_HOME`. |
| Texto no modo OpenCode | `internal/opencode.Client` → bridge HTTP com token em loopback → sessão do projeto permitido. A resposta volta direto ao WhatsApp. |
| Transcrição PTT no modo OpenCode | Sessão OpenCode existente; sem fallback para GPT se o bridge estiver indisponível. |

O SQLite guarda estado do dispositivo, modo, vínculos com sessões e IDs de
mensagem por conta, mais o ID da thread Codex. Notas PTT de até 120 segundos e
15 MiB são transcritas via comando local ou endpoint STT HTTP de loopback; o
áudio é descartado e a transcrição não entra no SQLite do Jarvis. O canal não
integra o histórico do Core. A sessão OpenCode tem contexto próprio e pode
enviar dados ao modelo configurado nela. A API do Core, o frontend e o
Tailscale são independentes do canal WhatsApp. Retenção/exclusão da thread,
resposta falada e piloto real seguem pendentes no
[roadmap](ROADMAP.md#prioridade-atual--jarvis-pessoal-no-whatsapp).

GPT/Codex permanece sem ferramentas locais de filesystem ou shell. OpenCode é o
único executor local para desenvolvimento. Integrações de trabalho externo,
como campanhas, devem usar APIs/MCPs específicos com autorização limitada; o
OAuth do Codex não disponibiliza automaticamente conectores do ChatGPT Work.

### Entradas planejadas e fronteira de atuação

As notas de voz atuais seguem o modo previamente confirmado por `/code`; voz
não ativa modo nem seleciona executor por inferência. A futura invocação
de IoT, digitada como `/cmd` ou falada explicitamente, seleciona o catálogo
local para um único turno. Comandos exatos podem ser resolvidos diretamente,
e o Ollama local pode interpretar linguagem natural em formato estruturado.
A proposta passa pela autorização de conta e pela validação de dispositivo,
operação e parâmetros no Core antes de chegar ao executor físico. GPT e
OpenCode continuam com suas rotas próprias e sem autoridade para chamar esse
executor. O módulo de imagens pertence ao chat normal e trata conteúdo
extraído como dados, sem interpretá-lo como comandos de atuação.

`/cmd` ainda não está implementado no WhatsApp. O trabalho foi adiado até haver
uma ponte Android real para as Google Home APIs; não automatizar a interface do
app Google Home como substituto dessa integração.


## Documentação das fases anteriores

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
│   │   ├── middleware/
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
├── voice-agent/
│   ├── src/jarvis_voice_agent/
│   ├── tests/
│   ├── pyproject.toml
│   └── README.md
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
- `backend/internal/middleware`: middlewares HTTP compartilhados; contem CORS.
- `backend/internal/services`: casos de uso e regras de aplicacao; contem chat com tool calling e tools iniciais.
- `backend/internal/repositories`: reservado para persistencia de dominio.
- `backend/internal/ai`: contrato `LLM` e provedores de IA; contem o provider inicial Ollama.
- `backend/internal/memory`: reservado para memoria persistente.
- `backend/internal/tools`: contrato `Tool`, registry, TimeTool, CalculatorTool e SystemTool.
- `backend/internal/voice`: contratos `SpeechRecognizer` e `VoiceProvider`, provider STT por comando, provider STT HTTP local e provider TTS por comando.
- `backend/internal/agents`: reservado para agentes autonomos.
- `backend/internal/voice`: reservado para voz.
- `backend/migrations`: scripts SQL de schema.
- `backend/docker`: Dockerfile do backend.

### Frontend

- `frontend/app`: rotas e layouts do Next.js App Router.
- `frontend/components`: componentes reutilizaveis.
- `frontend/services`: chamadas para APIs.
- `frontend/types`: tipos TypeScript compartilhados.
- `frontend/services/realtime-voice.ts`: cliente do `voice-agent` e do `livekit-client` para criar sessao, entrar na sala e publicar o microfone.
- `frontend/types/realtime-voice.ts`: contrato de sessao LiveKit usado pelo frontend.
- `frontend/hooks`: reservado para hooks.
- `frontend/public`: reservado para assets publicos.

### Voice Agent

- `voice-agent`: fronteira nova para voz realtime, separada do backend Go.
- `voice-agent/src/jarvis_voice_agent/jarvis_core.py`: cliente do Jarvis Core via `POST /api/chat/stream`.
- `voice-agent/src/jarvis_voice_agent/server.py`: servidor local minimo para health check e smoke test de chat.
- `voice-agent/src/jarvis_voice_agent/console.py`: modo console para validar streaming core sem audio.
- `voice-agent/src/jarvis_voice_agent/livekit.py`: emissor de tokens LiveKit para clientes desktop/Android.
- `voice-agent/src/jarvis_voice_agent/realtime.py`: participante LiveKit opcional que assina audio da sala, segmenta fala e chama STT/Jarvis Core.
- `voice-agent/src/jarvis_voice_agent/audio_turn.py`: VAD simples por RMS/silencio para formar turnos de fala.
- `voice-agent/src/jarvis_voice_agent/stt.py`: cliente HTTP multipart para o servidor STT local.
- `voice-agent/src/jarvis_voice_agent/wake.py`: deteccao de wake word por transcricao.

## Fluxos Entre Modulos

### Inicializacao do Backend

1. `cmd/api/main.go` chama `config.Load`.
2. `config.Load` le env vars e aplica defaults.
3. `database.New` cria `pgxpool.Pool` e executa `Ping`.
4. `ai.NewOllamaProvider` cria o provider configurado por `OLLAMA_URL` e `OLLAMA_MODEL`.
5. `services.NewChatService` recebe o contrato `ai.LLM`.
6. `tools.NewRegistry` cria o registry de ferramentas.
7. `TimeTool`, `CalculatorTool` e `SystemTool` sao registradas no registry.
8. `services.NewToolService` recebe o `ToolRegistry`.
9. Providers de STT/TTS sao criados a partir da configuracao; STT pode usar comando local ou servidor HTTP local persistente.
10. `services.NewVoiceService` recebe recognizer, provider e `ChatService`.
11. `handlers.NewRouter` cria `gin.Engine`.
12. `NewHealthHandler` recebe `database.Postgres`.
13. `NewChatHandler` recebe `ChatService`.
14. `NewToolHandler` recebe `ToolService`.
15. `NewVoiceHandler` recebe `VoiceService`.
16. Servidor HTTP inicia com shutdown gracioso.

### Health Check

1. Cliente chama `GET /health`.
2. `HealthHandler.Check` cria contexto com timeout de 2 segundos.
3. Handler chama `db.Ping`.
4. Se o banco responder, retorna `200` com `status: ok`.
5. Se o banco falhar, retorna `503` com `status: degraded`.

### Chat Por Texto Sem Streaming

1. Cliente chama `POST /api/chat` com `{ "message": "..." }`.
2. `ChatHandler` valida JSON, mensagem obrigatoria e tamanho maximo.
3. Handler cria contexto com timeout de 60 segundos.
4. `ChatService` verifica se a mensagem corresponde a uma tool obvia.
5. Se nenhuma tool for necessaria, `ChatService` chama `ai.LLM`.
6. Se uma tool for necessaria, `ChatService` executa a tool pelo `ToolService`.
7. `ToolService` delega a execucao ao `ToolRegistry`.
8. Tools simples retornam respostas deterministicas sem chamada extra ao LLM.
9. Backend retorna `{ "response": "..." }`.
10. Erros de validacao retornam `400`; falhas do LLM retornam `502`.

### Chat Por Texto Com Streaming

1. Cliente chama `POST /api/chat/stream` com `{ "message": "..." }`.
2. `ChatHandler` valida JSON, mensagem obrigatoria e tamanho maximo.
3. Handler responde com `Content-Type: text/event-stream`.
4. Se nenhuma tool for necessaria, `ChatService` chama `LLM.Stream`.
5. Se uma tool for necessaria, `ChatService` envia a resposta deterministica como evento SSE `delta`.
6. Ao terminar, backend envia evento SSE `done`.
7. Falhas durante a geracao viram evento SSE `error`.

### Tools

1. Cliente chama `GET /api/tools`.
2. `ToolHandler.List` chama `ToolService.ListTools`.
3. `ToolService` lista metadados pelo `ToolRegistry`.
4. Backend retorna um array com `name` e `description`.
5. Cliente chama `POST /api/tools/execute` com `{ "tool": "...", "input": "..." }`.
6. `ToolHandler.Execute` valida JSON, nome da ferramenta e tamanho do input.
7. `ToolService.ExecuteTool` delega ao `ToolRegistry`.
8. `ToolRegistry` busca a ferramenta registrada por nome e chama `Execute`.
9. Backend retorna `{ "result": "..." }`.
10. Ferramenta inexistente retorna `404`; input invalido da calculadora retorna `400`.

O fluxo manual de tools permanece disponivel para desenvolvimento e debug. No chat, a decisao automatica de uso de tool acontece no `ChatService` por roteamento local de casos obvios.

### SystemTool

1. O LLM ou o endpoint manual solicita a tool `system`.
2. O input recebido deve ser o nome de uma acao permitida.
3. `SystemTool` busca a acao em uma lista pre-cadastrada.
4. Se a acao nao existir, retorna erro de input invalido.
5. Se existir, executa apenas o comando e argumentos definidos no codigo para aquela acao.

`SystemTool` nao executa comandos arbitrarios, nao interpreta shell e nao usa a entrada do usuario como comando.

### Voz

1. Cliente chama `POST /api/voice/conversation` com multipart `audio`.
2. `VoiceHandler` valida upload, tamanho maximo, idioma e `wakeRequired`.
3. `VoiceService` chama `voice.SpeechRecognizer`.
4. O provider de STT pode chamar um servidor HTTP local persistente ou executar um comando local com arquivo temporario.
5. Se `wakeRequired=true` e a transcricao nao contiver a wake word, o backend retorna `ignored=true` sem chamar LLM, tools ou TTS.
6. Se houver wake word ou se `wakeRequired=false`, a transcricao e enviada para `ChatService`.
7. `ChatService` pode usar LLM e tools pelo `ToolRegistry`.
8. `VoiceService` chama `voice.VoiceProvider` para gerar audio da resposta.
9. O provider local de TTS executa o comando configurado e le o arquivo de audio gerado.
10. Backend retorna transcript, resposta textual, flags de wake word, content type do audio e audio em base64.

Os providers de voz usam processamento local configurado por env vars e nao dependem de APIs pagas por padrao. A troca futura de tecnologia deve implementar os mesmos contratos. O audio bruto nao deve ser persistido por padrao.

### Voice Agent Realtime

1. Cliente desktop ou Android abre uma sessao realtime com o agente de voz.
2. O cliente solicita `POST /livekit/session` no `voice-agent` para receber URL, sala, identidade e token LiveKit.
3. O transporte alvo e WebRTC via LiveKit, com pipeline Pipecat para VAD, STT, LLM e TTS streaming.
4. Apos transcrever uma fala, o agente chama `POST /api/chat/stream` no Jarvis Core.
5. O Jarvis Core continua responsavel por LLM, tools, permissoes, memoria futura e comandos do sistema.
6. O agente transforma os deltas SSE em texto para TTS streaming.
7. O audio de resposta volta pelo transporte realtime.

O slice implementado em `voice-agent` valida a conexao com o Jarvis Core sem audio, por console e HTTP local, emite tokens LiveKit para clientes desktop/Android e possui um participante realtime que assina audio da sala, segmenta turnos, transcreve via STT local e chama o Jarvis Core quando a wake word e detectada. O endpoint batch `POST /api/voice/conversation` permanece como fallback e caminho de debug, nao como arquitetura principal para experiencia natural.

### Frontend

1. `app/page.tsx` renderiza a interface principal de voz.
2. `VoiceAssistant` arma/desarma a escuta por botao.
3. `VoiceAssistant` captura audio com `MediaRecorder`.
4. `VoiceAssistant` usa VAD local com `AnalyserNode` para detectar inicio/fim de fala.
5. `sendVoiceConversation` envia `POST /api/voice/conversation` para `NEXT_PUBLIC_API_URL` com `wakeRequired=true`.
6. Se o backend retornar `ignored=true`, a interface volta ao standby sem resposta falada.
7. `VoiceAssistant` reproduz audio de resposta e anima `Speaking` com volume da reproducao.
6. `ChatPanel` permanece recolhido em `Modo Texto`.
7. `ToolsPanel` permanece recolhido para desenvolvimento e debug.
8. `HealthPanel` chama `getHealth`.
9. `getHealth` usa `API_INTERNAL_URL` no servidor Next.js.

### Docker Compose

1. `postgres` sobe com healthcheck.
2. `backend` espera `postgres` ficar healthy.
3. `frontend` depende do `backend` iniciado.
4. `voice-agent` roda como servico separado e depende do `backend`.
5. `livekit` fica disponivel no profile opcional `realtime` para transporte WebRTC local.

## Dependencias Existentes

Backend diretas:

- Gin para HTTP.
- pgx/pgxpool para PostgreSQL.
- Biblioteca padrao `net/http` para chamadas ao Ollama.
- Biblioteca padrao `os/exec` para providers locais de voz configuraveis.
- Biblioteca padrao `net/http` para provider STT HTTP local.

Voice Agent diretas:

- Python 3.11+.
- Biblioteca padrao para cliente HTTP/SSE e servidor local de smoke test.

Voice Agent opcionais:

- Pipecat para pipeline realtime.
- LiveKit para transporte WebRTC desktop/mobile.

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

## Sugestoes Arquiteturais Futuras

- Evoluir contratos entre handlers, services e repositories conforme conversa e memoria crescerem.
- Manter handlers sem dependencia direta de provedores externos de IA.
- Definir contratos para `memory` e `agents` antes da primeira implementacao concreta.
- Manter novas tools atras do contrato `Tool` e do `ToolRegistry`.
- Manter providers de voz batch atras dos contratos `SpeechRecognizer` e `VoiceProvider`.
- Manter voz realtime isolada no `voice-agent`, chamando o Jarvis Core em vez de executar tools diretamente.
- Manter migrations aditivas e aplicação versionada em runtime; usar
  `/docker-entrypoint-initdb.d` somente como bootstrap de banco novo.
- Continuar ampliando testes, sobretudo contratos com serviços reais e fluxos de
  reconexão; a quantidade de testes não substitui qualificação operacional.
- Manter erros HTTP consistentes nas novas rotas.
- Acesso remoto depende de grants/ACLs Tailscale por dispositivo; validar antes
  de habilitar uso móvel. Não há login Jarvis nesta trilha.
- Manter separacao entre URL interna de container e URL publica de navegador.

## Observacoes de Risco — rotas legadas e validações pendentes

- O bootstrap SQL do Compose só inicializa volume novo; o Core também aplica
  migrations versionadas em runtime ao iniciar.
- O backend falha ao iniciar se o banco nao estiver disponivel.
- O health check ainda acessa o wrapper de banco diretamente.
- As rotas legadas `/api/chat` e `/api/chat/stream` não compartilham a persistência
  conversacional de `/api/v2`.
- Tool calling inicial suporta uma chamada de tool por interacao; fluxos multi-step ficam para evolucao futura.
- STT/TTS dependem de binarios e modelos locais configurados no ambiente.
- O endpoint de voz batch ainda usa upload do trecho detectado por VAD; a direcao principal para experiencia natural passa a ser o `voice-agent` realtime.
- Wake word atual e baseada na transcricao, nao em keyword spotting acustico dedicado.
- A escrita OpenCode passou testes com mocks, mas ainda falta teste ponta a ponta
  contra serviço OpenCode real e acesso por telefone.
- `npm audit --omit=dev` aponta vulnerabilidades moderadas em dependencia transitiva do Next.js/PostCSS na versao observada.

## 2026-11-13 — Laboratório de voz contínua

A rota `/lab` testa uma sessão SmallWebRTC privada com o serviço Python
`jarvis_voice_agent.continuous`. Captura/reprodução e confirmação estimada de
reprodução ficam no Next.js; STT, VAD, Smart Turn e TTS ficam no Python; contexto,
IA e ferramentas ficam no Go. O PostgreSQL passa a persistir conversas, turnos,
eventos ordenados e execuções de ferramentas pela migração aditiva 002.

Os endpoints `/api/v2` coexistem com `/api/chat`, `/api/chat/stream` e o fallback
batch. A autoridade para histórico e ferramentas é o Core, não o agente Python
nem o provedor remoto. O novo fluxo só expõe horário e calculadora. O sistema
operacional fica bloqueado por padrão, inclusive no bootstrap legado.

A sessão usa identidade de turno e geração de áudio para invalidar saída antiga;
cancelamento reporta descarte local quando não há confirmação externa. Reconexão
conserva a conversa e cancela operações pendentes, sem reexecutar o turno antigo.
Somente o par STT/TTS selecionado fica carregado. Perfis, contratos, comandos,
limitações, licença Pocket pendente e validações de notebook estão descritos em
`docs/voice-lab/README.md`.

HTTPS/sinalização são servidos na mesma origem via Next.js/Tailscale Serve;
mídia WebRTC requer ICE/firewall próprios na tailnet. Não há exposição pública
nem dependência de serviço de mídia pago. Voz e inferência reais ainda dependem
dos testes de modelos/conta documentados; testes sintéticos não comprovam sua qualidade.
