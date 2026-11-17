# Adaptador local OpenCode do Jarvis

Adaptador local da integração Jarvis ↔ OpenCode V2. O serviço usa o cliente
oficial `@opencode/client` e descobre um serviço OpenCode já em execução no
computador; nunca inicia nem encerra esse serviço.

## Segurança atual

- Faz bind somente em `127.0.0.1`; não é um endpoint para o celular.
- Exige bearer token aleatório com pelo menos 32 bytes para toda rota funcional.
- Projetos vêm de allowlist explícita; paths relativos, ausentes e IDs duplicados
  são rejeitados. O path canônico nunca é devolvido pela API.
- A API expõe somente status do serviço, nomes dos projetos configurados e
  metadados mínimos de sessões daquele projeto. Não retorna conteúdo de mensagens.
- O Core Go consome o adaptador somente quando `JARVIS_OPENCODE_BRIDGE_ENABLED=true`;
  a tela `/code` lista os dados pela API do Core. Não há login Jarvis; o bearer
  token é somente interno entre Core e bridge e não deve ser tratado como login
  do usuário.
- Criação de sessões, prompts e decisões de permissão ficam desligados por padrão
  e exigem `JARVIS_OPENCODE_WRITE_ENABLED=true` tanto no Core quanto no bridge.
- Sessões usam a allowlist do projeto e pedem aprovação para todas as ferramentas.
  O Jarvis só oferece `once` e `reject`; não oferece `always`.

## Configuração e execução manual

Use Node.js 20 ou superior. Gere um token local:

```sh
python -c 'import secrets; print(secrets.token_urlsafe(32))'
```

O bridge pode ser iniciado sem login de usuário. Restrinja o acesso remoto ao
frontend pela tailnet; Core e bridge devem continuar em loopback. O token do
bridge deve ser o mesmo em `JARVIS_OPENCODE_BRIDGE_TOKEN` do Core e do serviço:

```sh
export JARVIS_OPENCODE_BRIDGE_TOKEN='token-gerado-acima'
export JARVIS_OPENCODE_PROJECTS='[{"id":"jarvis","name":"Jarvis","directory":"/caminho/absoluto/para/jarvis"}]'
export JARVIS_OPENCODE_BRIDGE_ENABLED=true
# Deixe false até concluir a configuração privada e o teste com projeto descartável.
export JARVIS_OPENCODE_WRITE_ENABLED=false
npm ci --prefix opencode-bridge
npm run build --prefix opencode-bridge
python scripts/jarvis run --stt fixture --tts tone --llm scripted
```

O `scripts/jarvis` inicia o adaptador opcionalmente junto do Core e frontend. O
bridge e as rotas Core ficam em loopback no fluxo do script.

O OpenCode V2 precisa estar instalado e com serviço local saudável. O adaptador
procura uma versão `2.x`; não inicia nem encerra o serviço. Rotas:

- `GET /healthz` — liveness local, sem detalhes do OpenCode.
- `GET /v1/status` — requer bearer token; versão e disponibilidade.
- `GET /v1/projects` — requer bearer token; allowlist sem diretórios.
- `GET /v1/projects/{id}/sessions` — requer bearer token; até 50 sessões, sem
  mensagens ou paths.
- `POST /v1/projects/{id}/sessions` — somente com escrita habilitada; cria uma
  sessão com permissões `ask` e registra o ID da operação em metadados para
  reconciliação.
- `POST /v1/projects/{id}/sessions/{session}/prompts` — somente com escrita
  habilitada; texto simples limitado a 8.000 caracteres. O serviço verifica o
  diretório canônico da sessão antes de enviar.
- `GET /v1/projects/{id}/sessions/{session}` — texto user/assistant, diff limitado
  e pedidos pendentes; reasoning, payloads de ferramentas e paths absolutos são
  removidos.
- `GET /v1/projects/{id}/operations/{id}` — reconcilia criação de sessão por
  metadados persistidos no OpenCode.
- `POST /v1/projects/{id}/sessions/{session}/permissions/{request}/reply` — aceita
  somente decisão `once` ou `reject`, após validar a sessão/projeto.

Rotas do Core: `GET /api/opencode/status`, `/projects`,
`/projects/{id}/sessions`, `/projects/{id}/jarvis-sessions`, `/sessions/{id}`;
POST para criar sessão, enviar prompt e responder à permissão. Escritas ficam
desabilitadas sem `JARVIS_OPENCODE_WRITE_ENABLED=true` e todas são registradas
antes do efeito externo. A porta do Core continua inacessível ao cliente remoto;
use Tailscale para limitar quem alcança o frontend. A UI está em `/code`.

## Desenvolvimento

```sh
npm test --prefix opencode-bridge
npm run build --prefix opencode-bridge
```
