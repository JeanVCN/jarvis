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

Este projeto ainda esta em desenvolvimento. A fase atual inclui a fundacao tecnica e a primeira integracao funcional de chat por texto com Ollama.

```text
Voce  ->  Navegador  ->  Jarvis API  ->  Memoria / IA / Ferramentas
                           |
                           v
                      PostgreSQL
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

O projeto esta na **Fase 2: Integracao inicial com Ollama**.

Ja existe:

- ✅ Backend em Go com Gin
- ✅ Frontend em Next.js, TypeScript e Tailwind
- ✅ PostgreSQL via Docker Compose
- ✅ Endpoint `GET /health`
- ✅ Endpoint `POST /api/chat`
- ✅ Endpoint `POST /api/chat/stream` com SSE
- ✅ Health check validando conexao com o banco
- ✅ Integracao inicial com Ollama via backend
- ✅ Tela simples de teste de chat com resposta em tempo real
- ✅ Configuracao por environment variables
- ✅ Estrutura preparada para IA, voz, memoria, ferramentas e agentes
- ✅ Documentacao arquitetural viva

Ainda nao existe:

- ❌ Voz
- ❌ Agentes autonomos
- ❌ Sistema de ferramentas
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

`localhost` tambem funciona, mas `127.0.0.1` e o padrao recomendado no projeto para evitar atrasos de resolucao entre IPv6/IPv4 em alguns ambientes locais.

Para o chat funcionar, mantenha o Ollama rodando e o modelo configurado em `OLLAMA_MODEL` disponivel localmente.

## ▶️ Rodando Pelo VS Code

O repositorio inclui configuracao para iniciar o ambiente pelo F5.

1. Abra o projeto no VS Code.
2. Abra a aba Run and Debug.
3. Selecione `Jarvis Full Stack`.
4. Pressione F5.

Esse perfil executa:

- Para containers `backend` e `frontend` do Docker Compose, caso estejam rodando.
- PostgreSQL via `docker compose up -d postgres`.
- Backend Go em modo debug na porta `18080`.
- Frontend Next.js com `npm run dev` na porta `13000`.

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

## 🔁 Portas do Projeto

Por padrao, o Jarvis usa portas menos comuns para evitar conflito com outros projetos locais:

- Frontend: `13000`
- Backend: `18080`
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
