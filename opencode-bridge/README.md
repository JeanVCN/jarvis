# Adaptador local OpenCode do Jarvis

Protótipo inicial **somente leitura** da integração C1. O serviço usa o cliente
oficial `@opencode/client` e descobre um serviço OpenCode V2 já em execução no
computador. Ele não inicia o OpenCode e não oferece endpoint para prompt,
comando, alteração de arquivo ou aprovação.

## Segurança atual

- Faz bind somente em `127.0.0.1`; não é um endpoint para o celular.
- Exige bearer token aleatório com pelo menos 32 bytes para toda rota funcional.
- Projetos vêm de allowlist explícita; paths relativos, ausentes e IDs duplicados
  são rejeitados. O path canônico nunca é devolvido pela API.
- A API expõe somente status do serviço, nomes dos projetos configurados e
  metadados mínimos de sessões daquele projeto. Não retorna conteúdo de mensagens.
- O Core Go consome o adaptador somente quando `JARVIS_OPENCODE_BRIDGE_ENABLED=true`;
  a tela `/code` lista os dados pela API do Core. A autenticação do produto é
  separada do bearer token interno do bridge.

## Configuração e execução manual

Use Node.js 20 ou superior. Gere um token local:

```sh
python -c 'import secrets; print(secrets.token_urlsafe(32))'
```

Configure `JARVIS_AUTH_REQUIRED=true`, senha/segredo do Jarvis, o token do bridge
e a lista JSON de projetos no ambiente (não versione esses valores). O token do
bridge deve ser o mesmo em `JARVIS_OPENCODE_BRIDGE_TOKEN` do Core e do serviço:

```sh
export JARVIS_OPENCODE_BRIDGE_TOKEN='token-gerado-acima'
export JARVIS_OPENCODE_PROJECTS='[{"id":"jarvis","name":"Jarvis","directory":"/caminho/absoluto/para/jarvis"}]'
export JARVIS_OPENCODE_BRIDGE_ENABLED=true
export JARVIS_AUTH_REQUIRED=true
npm ci --prefix opencode-bridge
npm run build --prefix opencode-bridge
python scripts/jarvis run --stt fixture --tts tone --llm scripted
```

O `scripts/jarvis` inicia o adaptador opcionalmente junto do Core e frontend. Os
segredos de autenticação Jarvis também precisam estar exportados. O bridge e as
rotas Core ficam em loopback no fluxo do script.

O OpenCode V2 precisa estar instalado e com serviço local saudável. O adaptador
procura uma versão `2.x`; não inicia nem encerra o serviço. Rotas:

- `GET /healthz` — liveness local, sem detalhes do OpenCode.
- `GET /v1/status` — requer bearer token; versão e disponibilidade.
- `GET /v1/projects` — requer bearer token; allowlist sem diretórios.
- `GET /v1/projects/{id}/sessions` — requer bearer token; até 50 sessões, sem
  mensagens ou paths.

Rotas read-only do Core: `GET /api/opencode/status`, `/api/opencode/projects` e
`/api/opencode/projects/{id}/sessions`. Habilitá-las pelo script também exige
`JARVIS_AUTH_REQUIRED=true` e os segredos de login configurados no startup. Isso
é uma checagem de configuração: o Core não valida o cookie do usuário por request.
O frontend Next aplica a sessão e encaminha as chamadas same-origin; mantenha a
porta do Core inacessível ao cliente remoto. A UI está em `/code`; nenhum endpoint
de prompt existe ainda.

## Desenvolvimento

```sh
npm test --prefix opencode-bridge
npm run build --prefix opencode-bridge
```
