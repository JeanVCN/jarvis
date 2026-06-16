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

Este projeto ainda esta em desenvolvimento. A fase atual foca na fundacao: arquitetura, backend, frontend, banco de dados, Docker e documentacao viva para manter consistencia ao longo do tempo.

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

O projeto esta na **Fase 1: Fundacao**.

Ja existe:

- ✅ Backend em Go com Gin
- ✅ Frontend em Next.js, TypeScript e Tailwind
- ✅ PostgreSQL via Docker Compose
- ✅ Endpoint `GET /health`
- ✅ Health check validando conexao com o banco
- ✅ Configuracao por environment variables
- ✅ Estrutura preparada para IA, voz, memoria, ferramentas e agentes
- ✅ Documentacao arquitetural viva

Ainda nao existe:

- ❌ Chat com IA
- ❌ Integracao com Ollama
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

- 🌐 Frontend: http://localhost:3000
- ❤️ Health check: http://localhost:8080/health

## 🩺 Health Check

```bash
curl http://localhost:8080/health
```

Resposta esperada:

```json
{
  "status": "ok",
  "database": "ok",
  "service": "jarvis-backend"
}
```

## 🔁 Portas Alternativas

Se alguma porta local ja estiver em uso, sobrescreva apenas as portas publicadas:

```bash
BACKEND_PORT=18080 FRONTEND_PORT=13000 NEXT_PUBLIC_API_URL=http://localhost:18080 docker compose up --build
```

Nesse caso:

- 🌐 Frontend: http://localhost:13000
- ❤️ Health check: http://localhost:18080/health

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
