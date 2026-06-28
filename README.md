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

Este projeto ainda esta em desenvolvimento. A fase atual inclui a fundacao tecnica, conversa por texto com Ollama, infraestrutura de ferramentas, roteamento local de tools obvias e uma interface de voz local com VAD.

```text
Voce  ->  Navegador/APK  ->  Voice Agent  ->  Jarvis API  ->  Memoria / IA / Ferramentas
             |                                  |
             v                                  v
       fallback batch                      PostgreSQL
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

O projeto esta na **Fase 5: Interface de Voz e Conversacao Natural**.

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
- ❌ Chamadas multiplas de tools em uma mesma interacao
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

Pipecat entra na proxima fatia como pipeline de audio dentro do agente para consumir o audio publicado na sala. Mesmo nesse modelo, tools e comandos continuam no backend Go, nao no agente Python.

Para iniciar o participante realtime do Jarvis na sala:

```bash
cd voice-agent
python -m pip install -r requirements-realtime.txt
JARVIS_LIVEKIT_ROOM=jarvis-main python -m jarvis_voice_agent.realtime
```

Nesta fatia, ele entra como `jarvis-agent`, assina faixas de audio e registra frames recebidos. A transcricao desses frames fica para o proximo passo.

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
