# Jarvis

<div align="center">

```
     ██╗ █████╗ ██████╗ ██╗   ██╗██╗███████╗
     ██║██╔══██╗██╔══██╗██║   ██║██║██╔════╝
     ██║███████║██████╔╝██║   ██║██║███████╗
██   ██║██╔══██║██╔══██╗╚██╗ ██╔╝██║╚════██║
╚█████╔╝██║  ██║██║  ██║ ╚████╔╝ ██║███████║
 ╚════╝ ╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚═╝╚══════╝
```

### Um assistente pessoal em desenvolvimento, inspirado no JARVIS do Homem de Ferro.

![Status](https://img.shields.io/badge/status-em%20desenvolvimento-33d6a6)
![Backend](https://img.shields.io/badge/backend-Go%20%2B%20Gin-00ADD8)
![Frontend](https://img.shields.io/badge/frontend-Next.js%20%2B%20Tailwind-111111)
![Database](https://img.shields.io/badge/database-PostgreSQL-4169E1)
![Docker](https://img.shields.io/badge/docker-compose-2496ED)

</div>

---

## ✨ Visao

O **Jarvis** nasce como um assistente pessoal local, modular e evolutivo.

A ideia e construir, passo a passo, um sistema capaz de conversar por texto e voz, manter memoria persistente, executar ferramentas, coordenar agentes autonomos e rodar com baixo custo operacional em ambiente Linux.

Este projeto ainda esta em desenvolvimento. O fluxo principal de laboratório usa
Next.js `/lab`, agente Python SmallWebRTC, Core Go `/api/v2` e PostgreSQL para
conversa, eventos e journal. O fluxo batch de voz permanece como fallback; consulte
`ARCHITECTURE.md` para separar arquitetura vigente de caminhos legados.

```text
Navegador /lab -> agente Python (SmallWebRTC) -> Core Go /api/v2 -> PostgreSQL
       |                                      ↘ provedores / ferramentas autorizadas
       └── voz batch (fallback)
```

## 🧠 Objetivos de Longo Prazo

- 💬 Conversacao por texto
- 🎙️ Conversacao por voz
- 🧠 Memoria persistente
- 🛠️ Sistema de ferramentas
- 🤖 Sistema de agentes autonomos
- 🐧 Execucao local em Linux
- 💸 Baixo custo operacional
- 🦙 Integracao inicial com Ollama
- 🌐 Acesso via navegador desktop e celular
- 🔐 Possibilidade de acesso remoto via Tailscale
- 🔌 Suporte futuro para OpenAI e Gemini

## 🚧 Status Atual

O projeto dispõe do laboratório de voz `/lab` e conversa persistida `/api/v2`.
GPT-6-Luna via OAuth isolado concluiu resposta e calculadora reais em 2026-11-14;
qualidade/latência de voz em dispositivo físico continuam sujeitas a validação.

**Prioridade ativa:** concluir a fundação segura para acesso móvel e implementar
delegação de tarefas ao OpenCode V2. O painel `/code` já oferece sessão/prompt
quando o servidor habilita `JARVIS_OPENCODE_WRITE_ENABLED`; a opção permanece
desligada por padrão. A direção e as fases C0–C4 estão em
[`ROADMAP.md`](ROADMAP.md#prioridade-atual--integração-jarvis--opencode). A etapa
residencial/Home APIs está pausada até o usuário ter Android; o probe compilado e
seu estado estão registrados em [`docs/home-apis/README.md`](docs/home-apis/README.md).

Já existe um adaptador local OpenCode em `opencode-bridge`. O modo read-only lista
status/projetos/sessões. O modo de escrita cria sessões, envia prompts, recupera
respostas/diffs, aceita decisões `once`/`reject` e solicita interrupção de tarefa;
ele exige uma opção explícita
no servidor e registra operações no PostgreSQL antes de chamar o executor. O app
não tem conta ou login Jarvis; o acesso remoto é limitado à tailnet.
O fluxo de escrita ainda é experimental: foi exercitado com mocks, mas aguarda
validação contra OpenCode real e regras Tailscale por dispositivo antes do uso.

**Avaliação de prontidão (2026-11-16):** estamos próximos de um piloto técnico,
mas não de uso remoto já comprovado. Antes de chamar “programar à distância” de
pronto, falta validar acesso permitido/negado pela tailnet e completar pelo
telefone uma tarefa em projeto descartável, incluindo aprovação e reconciliação
após desconexão. Depois desse gate, a prioridade pode voltar à IoT quando houver
Android; hoje a trilha residencial tem apenas probe read-only, sem OAuth nem
descoberta real. O gate completo está em [`ROADMAP.md`](ROADMAP.md).

### Acesso privado (fundação C0)

O script executa frontend, Core e bridge em loopback. Para acesso remoto, publique
somente o frontend na tailnet usando Tailscale Serve e configure a política da
tailnet para permitir os dispositivos escolhidos. Exemplo no computador Jarvis:

```sh
tailscale serve --bg --https=443 http://127.0.0.1:13000
```

Antes de usar, confirme que um dispositivo permitido conecta e que outro
dispositivo não permitido recebe bloqueio. Não publique as portas do frontend,
Core ou bridge na interface pública nem configure ACLs amplas para todos os
membros da tailnet. O Core e o bridge não validam identidade de usuário; a
fronteira de acesso é a rede privada. O app também valida `Origin` em requests
que alteram dados.

Ja existe:

- ✅ Backend em Go com Gin
- ✅ Frontend em Next.js, TypeScript e Tailwind
- ✅ PostgreSQL via Docker Compose
- ✅ Endpoint `GET /health`
- ✅ Endpoint `POST /api/chat`
- ✅ Endpoint `POST /api/chat/stream` com SSE
- ✅ Endpoint `GET /api/tools`
- ✅ Endpoint `POST /api/tools/execute`
- ✅ Health check validando conexao com o banco
- ✅ Integracao inicial com Ollama via backend
- ✅ `ToolRegistry` com `TimeTool`, `CalculatorTool` e `SystemTool`
- ✅ Chat capaz de acionar tools obvias automaticamente via `ToolRegistry`
- ✅ Endpoint `POST /api/voice/conversation`
- ✅ Contratos de voz `SpeechRecognizer` e `VoiceProvider`
- ✅ Base `voice-agent` para evoluir voz realtime com Pipecat/LiveKit
- ✅ Cliente do Jarvis Core via `POST /api/chat/stream`
- ✅ Interface principal baseada em voz
- ✅ Tela simples de teste de chat com resposta em tempo real
- ✅ Tela simples de teste de ferramentas
- ✅ Configuracao por environment variables
- ✅ Estrutura preparada para IA, voz, memoria, ferramentas e agentes
- ✅ Documentacao arquitetural viva

Ainda nao existe:

- ❌ Agentes autonomos
- ❌ Controle residencial (o fluxo `/api/v2` já suporta rodadas de ferramentas,
  mas autoriza somente `time` e `calculator`)
- ❌ Wake word acustica dedicada
- ❌ Transporte LiveKit/Pipecat com audio realtime conectado
- ❌ APK Android
- ❌ Autenticacao

## 🧰 Stack

### Backend

- Go
- Gin
- PostgreSQL
- pgx
- Docker

### Frontend

- Next.js
- React
- TypeScript
- Tailwind CSS

### Infraestrutura

- Docker Compose
- PostgreSQL 16

### Voice Agent

- Python 3.11+
- Cliente SSE para o Jarvis Core
- Pipecat/LiveKit planejados como dependencias opcionais para audio realtime
- Environment variables

### IA Planejada

- Ollama inicialmente
- OpenAI futuramente
- Gemini futuramente

## 🗂️ Estrutura do Projeto

```text
jarvis/
├── backend/
│   ├── cmd/
│   ├── internal/
│   │   ├── ai/
│   │   ├── voice/
│   │   ├── memory/
│   │   ├── tools/
│   │   ├── agents/
│   │   ├── handlers/
│   │   ├── services/
│   │   └── repositories/
│   ├── migrations/
│   └── docker/
│
├── frontend/
│   ├── app/
│   ├── components/
│   ├── hooks/
│   ├── services/
│   └── types/
│
├── voice-agent/
│   ├── src/jarvis_voice_agent/
│   ├── tests/
│   └── README.md
│
├── docker-compose.yml
├── BRAIN.md
├── ROADMAP.md
├── CONTEXT.md
├── ARCHITECTURE.md
├── DECISIONS.md
└── CONTRIBUTING_AI.md
```

## 🚀 Rodando Localmente

Clone o projeto e crie o arquivo de ambiente:

```bash
cp .env.example .env
```

Suba os servicos:

```bash
docker compose up --build
```

Acesse:

- 🌐 Frontend: http://127.0.0.1:13000
- ❤️ Health check: http://127.0.0.1:18080/health
- 🎙️ Voice Agent: http://127.0.0.1:18100/health

`localhost` tambem funciona, mas `127.0.0.1` e o padrao recomendado no projeto para evitar atrasos de resolucao entre IPv6/IPv4 em alguns ambientes locais.

Para o chat funcionar, mantenha o Ollama rodando e o modelo configurado em `OLLAMA_MODEL` disponivel localmente.

Para voz funcionar, configure STT/TTS locais. O perfil F5 usa um servidor STT local persistente em `127.0.0.1:18081`, mantendo o modelo carregado entre falas. O caminho por comando continua disponivel para Docker ou debug:

```env
VOICE_STT_PROVIDER=command
VOICE_STT_URL=
VOICE_STT_STATUS_URL=
VOICE_STT_COMMAND=faster-whisper
VOICE_STT_ARGS={input} --language {language} --model {model} --output_format txt --output_dir {output_dir}
VOICE_STT_MODEL=small
VOICE_STT_LANGUAGE=pt
VOICE_TTS_COMMAND=piper
VOICE_TTS_ARGS=--model {voice} --output_file {output}
VOICE_TTS_VOICE=/caminho/para/voz-pt_BR.onnx
```

Para rodar o servidor STT persistente manualmente:

```bash
python -m pip install -r scripts/requirements-voice.txt
JARVIS_STT_MODEL=small scripts/jarvis-stt-server
```

E configure o backend com:

```env
VOICE_STT_PROVIDER=http
VOICE_STT_URL=http://127.0.0.1:18081/transcribe
VOICE_STT_STATUS_URL=http://127.0.0.1:18081/health
```

Tambem existe `scripts/jarvis-stt-sherpa` para usar Sherpa-ONNX pelo provider de comando. Ele exige modelos Whisper ONNX configurados por:

```env
SHERPA_WHISPER_ENCODER=/caminho/encoder.onnx
SHERPA_WHISPER_DECODER=/caminho/decoder.onnx
SHERPA_WHISPER_TOKENS=/caminho/tokens.txt
```

### Voice Agent Realtime

O caminho batch acima continua disponivel como fallback e debug. A direcao principal para uma conversa realmente usavel passa a ser o modulo `voice-agent`, que deve receber audio realtime e chamar o Jarvis Core por `POST /api/chat/stream`.

Smoke test sem audio:

```bash
cd voice-agent
python -m venv .venv
. .venv/bin/activate
python -m pip install -e .
JARVIS_CORE_URL=http://127.0.0.1:18080 jarvis-voice-console
```

Servidor local do agente:

```bash
cd voice-agent
PYTHONPATH=src JARVIS_CORE_URL=http://127.0.0.1:18080 python -m jarvis_voice_agent.server
```

Teste HTTP:

```bash
curl -X POST http://127.0.0.1:18100/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Que horas sao?"}'
```

Gerar token de sessao LiveKit para um cliente desktop ou Android:

```bash
curl -X POST http://127.0.0.1:18100/livekit/session \
  -H "Content-Type: application/json" \
  -d '{"room":"jarvis-main","identity":"desktop"}'
```

O HUD principal tambem possui um controle para conectar na sala realtime usando `NEXT_PUBLIC_VOICE_AGENT_URL`. Ao conectar, o frontend solicita um token ao `voice-agent`, entra na sala LiveKit e publica o microfone.

Para subir o servidor LiveKit local gratuito em modo de desenvolvimento:

```bash
docker compose --profile realtime up livekit
```

Pipecat ainda pode substituir ou melhorar a pipeline de audio dentro do agente. Mesmo nesse modelo, tools e comandos continuam no backend Go, nao no agente Python.

Para iniciar o participante realtime do Jarvis na sala:

```bash
cd voice-agent
python -m pip install -r requirements-realtime.txt
JARVIS_LIVEKIT_ROOM=jarvis-main python -m jarvis_voice_agent.realtime
```

Nesta fatia, ele entra como `jarvis-agent`, assina faixas de audio, segmenta fala por VAD simples, envia o trecho para o STT local e chama o Jarvis Core quando a transcricao contem a wake word configurada.

## ▶️ Rodando Pelo VS Code

O repositorio inclui configuracao para iniciar o ambiente pelo F5.

1. Abra o projeto no VS Code.
2. Abra a aba Run and Debug.
3. Selecione `Jarvis Full Stack`.
4. Pressione F5.

Esse perfil executa:

- Para containers `backend` e `frontend` do Docker Compose, caso estejam rodando.
- PostgreSQL via `docker compose up -d postgres`.
- Servidor STT local persistente na porta `18081`.
- Backend Go em modo debug na porta `18080`.
- Frontend Next.js com `npm run dev` na porta `13000`.

Para validar a nova arquitetura de voz, selecione `Jarvis Realtime Voice Stack`. Esse perfil sobe backend, frontend e o `voice-agent` local na porta `18100`.

O Ollama deve estar rodando localmente em `http://127.0.0.1:11434` com o modelo configurado em `OLLAMA_MODEL`.

Se a porta `18080` continuar ocupada, encerre sessoes antigas de debug do backend ou verifique processos locais usando essa porta.

## 🩺 Health Check

```bash
curl http://127.0.0.1:18080/health
```

Resposta esperada:

```json
{
  "status": "ok",
  "database": "ok",
  "service": "jarvis-backend"
}
```

## 💬 Chat

Streaming usado pelo frontend:

```bash
curl -N -X POST http://127.0.0.1:18080/api/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message":"Ola Jarvis"}'
```

Resposta em eventos SSE:

```text
event: delta
data: {"delta":"Ola"}

event: done
data: {"done":true}
```

Endpoint simples sem streaming:

```bash
curl -X POST http://127.0.0.1:18080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Ola Jarvis"}'
```

Resposta esperada:

```json
{
  "response": "Ola, como posso ajudar?"
}
```

## WhatsApp — texto, notas de voz e imagens em modo experimental

O canal usa seu próprio número, Go 1.26+, CGO com compilador C e o GPT
autenticado pelo Codex App Server. Prepare uma vez o [perfil Codex do
Jarvis](docs/voice-lab/README.md) com login
no ChatGPT e mantenha suas variáveis exportadas. Na raiz do repositório:

```sh
cd backend/whatsapp
JARVIS_WHATSAPP_OWNER=5544XXXXXXXXX go run .
```

Use o número do aparelho que será vinculado, com país e DDD. Em **Aparelhos
conectados**, escaneie o QR nessa conta. Espere aparecer `WhatsApp conectado`
e mande texto comum no chat **Você**. Só esse chat do número configurado
recebe resposta. O nome configurado em `JARVIS_WHATSAPP_NAME` (padrão
`Jarvis`) orienta a personalidade, mas não aparece como prefixo. As respostas
no modo normal ficam sem rótulo; no modo `/code`, a última parte termina com
`*OpenCode*`.

O modelo segue o padrão do perfil Codex. Para selecionar outro modelo
elegível para sua conta, defina `JARVIS_WHATSAPP_MODEL`. O chat normal mantém
uma thread Codex persistente por conta vinculada. O Jarvis guarda somente o ID
dessa thread no SQLite; o histórico do modelo fica no `CODEX_HOME` do perfil
Codex. Personalize o contexto inicial com `JARVIS_WHATSAPP_PERSONA` e defina
`JARVIS_WHATSAPP_NAME` para dar nome próprio à instância.

Notas de voz (PTT) de até 120 segundos e 15 MiB são baixadas em memória e
transcritas pelo STT local configurado em `VOICE_STT_COMMAND` e
`VOICE_STT_ARGS`. O arquivo temporário usado pelo reconhecedor é removido ao
fim da chamada; o Jarvis não grava áudio nem transcrição no SQLite. A
transcrição segue o modo já confirmado no chat: modo normal envia texto ao GPT
pela thread persistente; `/code` envia texto à sessão OpenCode existente. A
transcrição não pode ativar `/code` por conta própria e uma falha no OpenCode
nunca redireciona o conteúdo ao GPT. Sem STT local disponível, o Jarvis pede
outro áudio ou texto. Mensagens com áudio que não sejam notas PTT continuam
ignoradas.

Fotos do WhatsApp no chat **Você** são encaminhadas no formato original,
sem OCR, redimensionamento ou classificação local. O MVP aceita JPEG, PNG e
WebP de até 10 MiB, com legenda opcional. A rota respeita o modo atual: no modo
normal a imagem segue inline ao Codex; em `/code`, segue como anexo inline à
sessão OpenCode. Não há fallback para outro provedor se o destino não aceitar
imagens. O Jarvis mantém os bytes apenas durante o encaminhamento, mas o Codex
pode reter a imagem na thread persistente e o OpenCode na própria sessão; o
modelo configurado no OpenCode pode ser remoto. Use apenas modelos com visão.

O SQLite também guarda as credenciais do dispositivo vinculado para reconectar.
Para rodar dois números ao mesmo tempo, inicie uma instância do processo para
cada aparelho, usando `JARVIS_WHATSAPP_OWNER` diferente em cada terminal.
Cada número tem pareamento, SQLite, thread Codex, nome e personalidade próprios.
As duas instâncias podem compartilhar o mesmo perfil autenticado do Codex
(`JARVIS_CODEX_HOME`), usando a mesma conta GPT, sem compartilhar a thread ou o
histórico de conversa do Jarvis. Configure `JARVIS_WHATSAPP_NAME` e
`JARVIS_WHATSAPP_PERSONA` separadamente em cada processo. Exemplo:

```sh
# Terminal 1: seu número e sua conversa pessoal
cd backend/whatsapp
JARVIS_WHATSAPP_OWNER=5544111111111 JARVIS_WHATSAPP_NAME=Jarvis go run .

# Terminal 2: outro aparelho com identidade própria
cd backend/whatsapp
JARVIS_WHATSAPP_OWNER=5544222222222 JARVIS_WHATSAPP_NAME=JarvisCasa \
  JARVIS_WHATSAPP_PERSONA='Ajude com rotinas e informações da casa.' go run .
```

Use o mesmo `JARVIS_CODEX_HOME` nos dois terminais para reutilizar a autenticação,
mas mantenha números distintos. Cada processo precisa continuar rodando; não
inicie duas instâncias para o mesmo número. Ao voltar a iniciar uma instância,
o Jarvis reutiliza o vínculo correspondente, se ainda estiver ativo. O `whatsmeow`
é uma integração não oficial, sujeita a desconexão ou restrição de conta.
O canal usa conexões de saída para WhatsApp e GPT. Para trocar mensagens,
basta o processo estar rodando no notebook com internet. Tailscale fica
para o acesso remoto à interface web e ao `/code`. O teste com uma conta
real continua pendente.

### Manter o Jarvis rodando no Linux

O QR do primeiro pareamento ainda aparece no terminal do processo WhatsApp; a
interface web não tem onboarding de pareamento. Faça esse pareamento uma vez
com `go run .` e, depois de ver `WhatsApp conectado`, encerre o processo. Evite
parear através do journal do systemd: o QR é uma credencial temporária e pode
ficar retido nos logs.

```sh
cd backend/whatsapp
JARVIS_WHATSAPP_OWNER=5544111111111 \
  JARVIS_CODEX_ENABLED=true JARVIS_CODEX_HOME="$HOME/.codex" go run .
```

Depois instale os serviços de usuário. O instalador compila um binário para cada
número e prepara unidades que iniciam no login, reiniciam após falha e enviam
logs ao journal. Ele não inicia nem habilita serviços automaticamente.

```sh
python3 scripts/install-systemd-user.py --owner 5544111111111
```

Para habilitar também o bridge OpenCode e o site/Core sem captura contínua de
voz, instale as dependências e compile frontend e bridge antes:

```sh
cd frontend
npm ci && npm run build
cd ../opencode-bridge
npm ci && npm run build
cd ..
python3 scripts/install-systemd-user.py \
  --owner 5544111111111 --with-opencode-bridge --with-web
```

Revise `~/.config/jarvis/whatsapp-NUMERO.env` e, se usar `/code`, configure
`~/.config/jarvis/opencode-bridge.env` com a allowlist JSON `JARVIS_OPENCODE_PROJECTS`,
`JARVIS_OPENCODE_BRIDGE_ENABLED=true` e `JARVIS_OPENCODE_WRITE_ENABLED=false`
até a validação local. O instalador gera
o token interno do bridge com permissões `0600`; o serviço WhatsApp recebe esse
mesmo token. O ID de projeto usado pela instância fica em
`JARVIS_WHATSAPP_OPENCODE_PROJECT` no arquivo dela. O perfil Codex padrão é
`~/.codex`; se usar outro, informe `JARVIS_CODEX_HOME` no env file da instância.
Esses arquivos são locais e não devem ser commitados.

Exemplo da allowlist (ajuste o caminho absoluto; mantenha a linha do token que
o instalador gerou):

```ini
JARVIS_OPENCODE_PROJECTS="[{\"id\":\"jarvis\",\"name\":\"Jarvis\",\"directory\":\"/home/usuario/jarvis\"}]"
JARVIS_OPENCODE_BRIDGE_ENABLED=true
JARVIS_OPENCODE_WRITE_ENABLED=false
```

Habilite somente os serviços que for usar:

```sh
systemctl --user enable --now jarvis-whatsapp@5544111111111.service
systemctl --user enable --now jarvis-opencode-bridge.service
systemctl --user enable --now jarvis-web.service
```

Pareamento, canal WhatsApp, bridge, Core e frontend podem ser consultados com
`systemctl --user status` e `journalctl --user -u NOME-DO-SERVICO`. Para iniciar
os serviços no boot antes de abrir a sessão gráfica, habilite o *linger* do
usuário uma vez com `loginctl enable-linger SEU-USUARIO`. A configuração de
energia do host é separada: bloqueie a sessão, deixe o display apagar e mantenha
o notebook acordado na tomada. Os units não bloqueiam suspensão.

O unit web usa `scripts/jarvis run --no-voice-agent`, reduzindo trabalho em
repouso. O serviço WhatsApp sozinho não precisa do Core/frontend; `/code`
precisa do bridge. Depois de atualizar o checkout, rode novamente o instalador
para recompilar os binários e atualizar os units. A instalação e o teste em um
Linux de usuário ainda precisam ser validados no notebook-alvo.

As respostas convertem títulos, listas, links, negrito, itálico, tachado,
blocos de código e tabelas para formas mais legíveis no Markdown do WhatsApp.
Respostas extensas são divididas em mensagens menores, preservando blocos de
código entre partes. Os IDs de mensagens enviadas pelo Jarvis são guardados
para que eventos de eco não voltem à conversa; o conteúdo das respostas não é
duplicado no SQLite.

### OpenCode no chat Você

Envie `/code` e espere a confirmação antes de enviar a tarefa. Envie
`/code` novamente para voltar ao GPT do chat normal. O modo e o vínculo da
sessão do OpenCode são persistidos por número e sobrevivem ao reinício.

Configure `JARVIS_WHATSAPP_OPENCODE_PROJECT` com o ID de um projeto da
allowlist do bridge, `JARVIS_OPENCODE_BRIDGE_TOKEN` com a credencial interna e,
se necessário, `JARVIS_OPENCODE_BRIDGE_URL` com o endereço HTTP de loopback
(padrão `http://127.0.0.1:18110`). Use o
[bridge local do OpenCode](docs/opencode-integration.md) com a escrita
habilitada para o ambiente de teste. O WhatsApp não precisa acessar o Core
ou o frontend. Se o OpenCode ficar indisponível, a mensagem não segue para
o GPT normal. Perguntas estruturadas do OpenCode chegam com o texto do agente,
títulos, alternativas, descrições e a marca `Recommended` quando fornecida.
Responda com o número da opção; texto que não seja um número é enviado como
resposta personalizada. Formulários com vários campos aceitam valores na
ordem, separados por ponto e vírgula. Para autorizações pendentes, use
`/aprovar` ou `/rejeitar`; se houver mais de um pedido, acrescente o número,
como `/aprovar 2`. A aprovação vale somente para a ação selecionada. A troca
de modo muda o destino de novas mensagens, sem cancelar tarefas já enviadas.
Resultados e contexto são tratados pelo provedor configurado no próprio
OpenCode.

## 🛠️ Tools

Listar ferramentas:

```bash
curl http://127.0.0.1:18080/api/tools
```

Resposta esperada:

```json
[
  {
    "name": "calculator",
    "description": "Executa calculos basicos"
  },
  {
    "name": "system",
    "description": "Executa acoes controladas do sistema operacional. Acoes permitidas: lock_screen, open_browser, open_spotify, open_terminal, open_vscode"
  },
  {
    "name": "time",
    "description": "Retorna data e hora atual"
  }
]
```

Executar ferramenta:

```bash
curl -X POST http://127.0.0.1:18080/api/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool":"calculator","input":"25 * 4"}'
```

Resposta esperada:

```json
{
  "result": "100"
}
```

Executar uma acao controlada do sistema:

```bash
curl -X POST http://127.0.0.1:18080/api/tools/execute \
  -H "Content-Type: application/json" \
  -d '{"tool":"system","input":"open_vscode"}'
```

O chat tambem pode acionar tools automaticamente. Exemplos de mensagens:

```text
Que horas sao?
Quanto e 25 * 4?
Abra o navegador
Abra o VSCode
```

O backend usa roteamento local para tools obvias, preservando streaming para mensagens normais e evitando chamada extra ao LLM em comandos simples.

## 🎙️ Voz

Checar se STT/TTS locais estao disponiveis para o backend:

```bash
curl http://127.0.0.1:18080/api/voice/status
```

Resposta esperada quando tudo esta configurado:

```json
{
  "sttAvailable": true,
  "ttsAvailable": true,
  "sttCommand": "http://127.0.0.1:18081/transcribe",
  "ttsCommand": "piper",
  "message": "voice_ready",
  "privacyMode": "local"
}
```

Endpoint de conversa por voz:

```bash
curl -X POST http://127.0.0.1:18080/api/voice/conversation \
  -F "audio=@fala.webm;type=audio/webm" \
  -F "language=pt"
```

Resposta esperada:

```json
{
  "transcript": "Que horas sao?",
  "response": "Agora sao 17:10.",
  "audioContentType": "audio/wav",
  "audioBase64": "...",
  "ignored": false,
  "wakeWordDetected": true
}
```

Na interface principal, o botao de voz arma a escuta. O navegador usa VAD local para detectar quando voce terminou de falar. Em modo armado, envie comandos como:

```text
Jarvis, que horas sao?
Jarvis, abra o Spotify
```

Se a transcricao nao contiver `Jarvis`, o backend retorna `ignored=true` e nao chama LLM, tools ou TTS.

Para uma experiencia natural, a evolucao recomendada e usar o `voice-agent` com audio realtime. O endpoint `/api/voice/conversation` permanece util para fallback, testes isolados de STT/TTS e comparacao de latencia.

Se STT/TTS local nao estiver instalado ou configurado, o backend retorna:

```json
{
  "error": "voice_provider_unavailable"
}
```

## 🔁 Portas do Projeto

Por padrao, o Jarvis usa portas menos comuns para evitar conflito com outros projetos locais:

- Frontend: `13000`
- Backend: `18080`
- Voice Agent: `18100`
- LiveKit: `7880`, `7881`, `7882/udp`
- STT local persistente: `18081`
- PostgreSQL: `5432`
- Ollama: `11434`

Esses valores estao em `.env.example`, `internal/config` e na configuracao de F5 do VS Code.
As URLs locais publicas usam `127.0.0.1` por padrao, enquanto o CORS aceita `127.0.0.1` e `localhost`.

## 🔁 Portas Alternativas

Se alguma porta local ja estiver em uso, sobrescreva apenas as portas publicadas:

```bash
BACKEND_PORT=28080 BACKEND_CONTAINER_PORT=18080 FRONTEND_PORT=23000 NEXT_PUBLIC_API_URL=http://127.0.0.1:28080 CORS_ALLOWED_ORIGIN=http://127.0.0.1:23000,http://localhost:23000 docker compose up --build
```

Nesse caso:

- 🌐 Frontend: http://127.0.0.1:23000
- ❤️ Health check: http://127.0.0.1:28080/health

## 📚 Documentacao Viva

Este projeto possui uma camada de memoria permanente para ajudar futuras sessoes de desenvolvimento, agentes de IA e colaboradores humanos.

- 🧠 `BRAIN.md`: memoria permanente do projeto
- 🗺️ `ROADMAP.md`: estado atual e proximas etapas
- 📌 `CONTEXT.md`: conceitos e contexto do sistema
- 🏛️ `ARCHITECTURE.md`: arquitetura atual
- 📝 `DECISIONS.md`: decisoes arquiteturais
- 🤝 `CONTRIBUTING_AI.md`: instrucoes para futuras IAs colaboradoras

Antes de alterar codigo, leia esses arquivos.

## 🧭 Filosofia

O Jarvis deve crescer com calma, clareza e consistencia.

Nada de implementar tudo de uma vez. A base foi preparada para que cada parte evolua com responsabilidade:

- primeiro a fundacao;
- depois conversa por texto;
- depois memoria;
- depois ferramentas;
- depois voz;
- depois agentes autonomos.

Pequenos passos, arquitetura limpa, documentacao viva e uma ambicao grande o bastante para valer a jornada.

---

<div align="center">

**Jarvis esta em desenvolvimento.**  
Fundacao hoje. Assistente inteligente amanha. 🚀

</div>

## Laboratório de voz contínua (novo)

A rota `/lab` preserva a interface atual e permite comparar Whisper base/small,
Kokoro/Pocket, silêncio/Smart Turn e motores de texto pelo mesmo fluxo. Instalação,
perfis, segurança, tailnet, testes e limitações comprovadas estão no
[guia de voz contínua](docs/voice-lab/README.md).

A execução real de modelos e contas ainda exige as verificações indicadas no guia.
Não há fallback pago. Não confunda os testes `fixture/tone` com reconhecimento ou síntese reais.
