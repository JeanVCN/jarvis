# Laboratório de voz contínua do Jarvis

Esta entrega adiciona `/lab`, mantendo a home e os endpoints anteriores. A configuração final será escolhida no i5-1135G7 / 8 GB / Linux, com Go, PostgreSQL e Next.js ativos. Nenhum resultado do executor remoto demonstra o desempenho desse notebook.

## Estado e limites comprovados

- Go: provedores em streaming, ferramentas seguras, histórico no PostgreSQL, eventos retomáveis, cancelamento local e journal de ferramentas.
- Python: Pipecat SmallWebRTC, Silero ONNX, Smart Turn v3.2 ONNX, fila de PCM limitada a 240 ms, fila de orações limitada, STT/TTS em workers separados e descarte por geração.
- STT: adaptadores faster-whisper `base`/`small`, multilíngues, CPU INT8, `pt`, dois threads e um worker.
- TTS: adaptadores Kokoro 0.9.4 (`pf_dora`, `pm_alex`, `pm_santa`) e Pocket TTS 4.0.1 (`portuguese`, `rafael`) pelo mesmo contrato. Somente o perfil selecionado é carregado. Não usamos o adaptador Pocket antigo do Pipecat, que depende de outra versão.
- Motores: scripted; Groq Free mediante confirmação; Ollama; Codex App Server experimental, mediante ativação e login em diretório isolado.
- Sem áudio bruto salvo por padrão. Exportações de métricas e corpus podem conter texto pessoal: ficam locais e ignorados pelo Git.

**Ainda não comprovados:** inferência de Kokoro/Pocket/Whisper com pesos reais neste executor; qualidade PT-BR; inferência em conta ChatGPT/Groq; cotas da conta; áudio no navegador físico; Android e mídia pela tailnet; metas de latência/RAM. O Chromium gerenciado do executor impõe `WebRtcIPHandling=disable_non_proxied_udp`; seu teste de mídia não pode ser concluído aqui. A interface foi exercitada separadamente e o transporte foi exercitado por um cliente aiortc com áudio sintético.

Os downloads Hugging Face e as páginas de Sign in with ChatGPT/Groq foram negados pela política de rede do executor (403 do proxy). Nenhuma verificação TLS foi desativada e nenhuma restrição do Chromium foi removida. `fixture`/`tone` são exclusivamente testes de transporte: **não são reconhecimento/síntese reais**.

## Preparar o notebook

Pré-requisitos: Linux x86_64, Python 3.12 com venv, Go >=1.25, Node 24, npm e Docker Compose. O script não instala pacotes do sistema, não ativa cobrança, não faz login e não altera a tailnet.

Na raiz do checkout existente:

```bash
cp .env.continuous.example .env.continuous
chmod 600 .env.continuous
set -a
. ./.env.continuous
set +a
./scripts/jarvis setup
```

`setup` instala PyTorch **CPU** antes das demais dependências, aplica as versões testadas como constraints e usa `npm ci`. Não baixe wheels CUDA para a Iris Xe. O caminho inicial é CPU; não há promessa de aceleração da GPU integrada.

Feche os processos antigos que ocupem 13000/18080/18100 antes de iniciar. Não execute dois Cores sobre o mesmo banco. O Core mantém um lock de instância no PostgreSQL e aplica migrações automaticamente. O script conserva o volume existente de PostgreSQL.

```bash
./scripts/jarvis prepare --stt base --tts kokoro --voice pf_dora
./scripts/jarvis run --stt base --tts kokoro --voice pf_dora --llm scripted
```

Abra a rota `/lab` no navegador local. Conectar arma uma sessão explícita de escuta contínua; o perfil novo não exige repetir a wake word em cada turno. Desconecte para encerrar a captura. O perfil escolhido no comando é lido pela interface. Os seletores permitem mudar uma variável por vez após desconectar; reconecte para carregar o novo perfil. `Ctrl+C` encerra os três processos iniciados pelo script; PostgreSQL permanece ativo.

Para Whisper small:

```bash
./scripts/jarvis prepare --stt small --tts kokoro --voice pf_dora
./scripts/jarvis run --stt small --tts kokoro --voice pf_dora --llm scripted
```

Kokoro usa `lang_code=p`. Pocket usa o modelo oficial de português, não o inglês nem um modelo comunitário com nome parecido. Antes de habilitar Pocket, leia os termos dos pesos e da amostra da voz `rafael`: a biblioteca remete ao áudio `g-Vi8PgmSY0-enhanced-v2.wav` no repositório `kyutai/pocket-tts`, revisão `64ab7d24c479d736a83b8cc666c4a776fca30fda`. A licença individual não pôde ser confirmada neste ambiente. A confirmação abaixo **não substitui essa revisão**:

```bash
export JARVIS_POCKET_LICENSE_ACCEPTED=true
./scripts/jarvis prepare --stt base --tts pocket --voice rafael
./scripts/jarvis run --stt base --tts pocket --voice rafael --llm scripted
```

Referências: [Kokoro](https://huggingface.co/hexgrad/Kokoro-82M), [Pocket TTS](https://github.com/kyutai-labs/pocket-tts), [licenças das vozes](https://huggingface.co/kyutai/tts-voices). A verificação de sotaque brasileiro continua manual; suporte a português não garante sotaque PT-BR.

Os primeiros carregamentos baixam pesos; meça aquecimento e primeira execução separadamente. Preserve o cache Hugging Face entre comparações. Registre os hashes/revisões dos artefatos baixados junto dos resultados; as dependências têm constraints, mas os downloads padrão de modelos ainda dependem das revisões publicadas pelo fornecedor.

## Motor real de IA

Não há fallback entre provedores. Na ausência de credencial, elegibilidade ou cota, a sessão reporta erro. As ferramentas automáticas da nova conversa são somente `calculator` e `time`; `system` não é oferecida ao modelo. Também desabilitamos seu registro no fluxo legado por padrão. A variável `JARVIS_ALLOW_SYSTEM_TOOLS=true` restaura explicitamente o comportamento legado; não é necessária para os testes.

### Groq Free

1. Use uma organização Free sem cobrança habilitada e verifique os limites atuais no console.
2. Guarde `GROQ_API_KEY` no arquivo local ignorado; nunca no navegador nem no Git.
3. Defina `JARVIS_GROQ_FREE_CONFIRMED=true` **no processo do Go**. A API não fornece aqui uma comprovação independente do plano: essa confirmação é operacional, não uma garantia técnica de faturamento.
4. Inicie o laboratório com `--llm groq --model openai/gpt-oss-120b`.
5. Execute a inferência e a ferramenta reais:

```bash
python3 scripts/check_providers.py --provider groq --model openai/gpt-oss-120b
python3 scripts/check_providers.py --provider groq --model qwen/qwen3.8-27b
```

Só compare um ID se ele estiver no catálogo ativo e passar no teste. A presença no catálogo não basta. A allowlist do adaptador limita os dois candidatos pesquisados; indisponibilidade resulta em erro e exige revisão explícita. `429` emite `quota_exhausted`, com `retry_after` quando informado; não repetimos ferramentas nem ativamos outro provedor. Quotas/dados: [limites](https://console.groq.com/docs/rate-limits), [tratamento de dados](https://console.groq.com/docs/your-data).

### ChatGPT via Codex App Server — experimental

O adaptador usa o protocolo stdio, sem extrair cookies/tokens e sem chamar a API convencional com chave. A validação de enquadramento do assistente pessoal nas condições de Sign in with ChatGPT permanece necessária: [fluxo oficial](https://developers.openai.com/siwc/token-sharing-open-source/codex-app-server), [self-hosted](https://developers.openai.com/siwc/token-sharing-open-source/self-hosted-vms).

Os schemas de `initialize`, `account/read`, `thread/start`, `turn/start` e eventos foram inspecionados no CLI 0.134.0; inferência real não foi executada, pois não há conta do usuário conectada. Não reutilize a autenticação do executor Codex.

Preparação local opcional:

```bash
mkdir -p .local/codex-profile
chmod 700 .local/codex-profile
npm install --prefix .local/codex-cli --no-audit --no-fund @openai/codex@0.134.0
export JARVIS_CODEX_HOME="$(pwd)/.local/codex-profile"
export JARVIS_CODEX_BIN="$(pwd)/.local/codex-cli/node_modules/.bin/codex"
CODEX_HOME="$JARVIS_CODEX_HOME" "$JARVIS_CODEX_BIN" login
export JARVIS_CODEX_ENABLED=true
```

Liste o catálogo com `python3 scripts/codex_models.py` (isso ainda não comprova elegibilidade). Reinicie o Go com essas variáveis, selecione `codex` e um candidato, e rode:

```bash
python3 scripts/check_providers.py --provider codex --model ID_DO_MODELO
```

O teste só passa com inferência concluída e resultado real da calculadora. Compare um modelo rápido e outro mais capaz se ambos passarem. O adaptador desabilita shell, unified exec, apply_patch, apps, multi-agent e web search, usa thread efêmera/read-only e recusa operações nativas notificadas. Pedidos de ferramentas Jarvis usam um envelope JSON restrito e passam pelo mesmo journal Go; essa integração ainda precisa de validação com modelo real, inclusive de sua latência.

Para trocar conta: interrompa a sessão e pare o Core, execute `logout` e `login` com o mesmo `CODEX_HOME`, depois reinicie. O histórico permanece no PostgreSQL, mas não é enviado à nova conta até você explicitamente continuar uma conversa. Use “Nova conversa” para separar contexto. O App Server é iniciado por rodada; não há conta/chave armazenada na web. Cancelamento externo é tentado, porém só reportamos descarte local; **não presumimos confirmação do provedor**.

### Ollama

O padrão continua `llama3.2`. Execute `ollama list` e `ollama show llama3.2` no notebook e registre digest, tamanho e quantização. O adaptador novo usa `/api/chat`, contexto de 2048 e dois threads. `llama3.2` recebe conversas sem ferramentas nativas, pois o modelo padrão pode rejeitá-las. Isso ficará evidente no teste de ferramentas; não o considere aprovado só por conversar.

A variante menor proposta para uma segunda referência é `qwen3:1.7b` Q4_K_M, com `think=false`, se o artefato estiver disponível e suportar ferramentas. Meça o conjunto LLM+STT+TTS antes de adotá-lo nos 8 GB.

## Tailnet e HTTPS

Processos HTTP escutam em loopback; as publicações Compose de Go, web, voz e PostgreSQL também são limitadas a loopback. Configure manualmente no notebook, sem Funnel:

```bash
tailscale serve --bg --https=443 http://127.0.0.1:13000
```

Inclua a origem HTTPS exata do seu node em `CORS_ALLOWED_ORIGIN` e `JARVIS_ALLOWED_ORIGINS` antes de iniciar. Web/API/sinalização usam a mesma origem; `/api/*`, `/health` e `/voice/*` são encaminhados pelo Next.js. URLs personalizadas de proxy precisam estar presentes também durante `npm run build`, pois rewrites de produção são gerados no build.

**Serve não transporta a mídia UDP.** O SmallWebRTC não oferece nesta integração uma faixa fixa de portas. Ele usa portas UDP efêmeras do sistema e anuncia interfaces encontradas. Para a tailnet:

1. Execute o serviço nativamente no Linux, com `tailscale0` visível; evite colocar mídia em uma rede Docker isolada nesta primeira rodada.
2. Configure `JARVIS_ICE_ADDRESS` com o IPv4 Tailscale do servidor. O serviço filtra os candidatos anunciados e falha com `ice_address_unavailable` se esse endereço não tiver sido coletado.
3. Restrinja no firewall o acesso às portas de mídia à interface `tailscale0` e aos dispositivos autorizados. O filtro de SDP não muda os sockets de escuta e não substitui o firewall.
4. Nas grants/ACLs da tailnet, autorize o IP Tailscale do dispositivo cliente para TCP 443 e UDP de mídia do servidor. Não autorize todos os membros por conveniência; preserve regras existentes.
5. Confirme em eventos/stats o par ICE selecionado e tráfego bidirecional. Se só houver sinalização, o teste falhou.
6. Teste Android com Wi-Fi e depois com Wi-Fi desligado/4G, mantendo Tailscale conectado.

Não abrimos portas do roteador nem instalamos TURN público. Se a combinação real de ICE/firewall/navegador falhar, a próxima comparação é LiveKit self-hosted, conservado como perfil legado; não afirmamos que ele já resolve a conectividade sem testar.

## Reproduzir checks remotos

Com Core e serviço contínuo iniciados, os testes abaixo criam conversas de teste persistidas:

```bash
cd backend
go test -race -p 2 ./...
cd ../voice-agent
PYTHONPATH=src .venv/bin/python -m pytest -q
cd ../frontend
npm run lint
npm run build
cd ..
python3 scripts/core_smoke.py
python3 scripts/check_providers.py --provider scripted
```

Para transporte sem baixar pesos, inicie **somente como perfil de teste**, com `JARVIS_TEST_AUDIO=true`, e execute:

```bash
./scripts/jarvis run --synthetic --stt fixture --tts tone --llm scripted
PYTHONPATH=voice-agent/src voice-agent/.venv/bin/python scripts/rtc_smoke.py
```

O cliente aiortc verifica áudio sintético recebido, deltas, interrupção, calculadora e retomada da identidade da conversa. Isso não comprova STT nem naturalidade. `JARVIS_TEST_CORE` e `JARVIS_TEST_VOICE` permitem portas alternativas nos smoke tests.

## Comparação no notebook

Copie `corpus.json` para uma pasta local e grave os arquivos WAV indicados com sua voz. Repita frases com fone, alto-falante, ruído e autocorreções. Não altere o corpus entre candidatos.

Comparação isolada, sem subir o pipeline todo:

```bash
voice-agent/.venv/bin/python scripts/voice_bench.py --corpus docs/voice-lab/corpus.json --stt base --output results/stt-base.json
voice-agent/.venv/bin/python scripts/voice_bench.py --corpus docs/voice-lab/corpus.json --stt small --output results/stt-small.json
voice-agent/.venv/bin/python scripts/voice_bench.py --corpus docs/voice-lab/corpus.json --tts kokoro --voice pf_dora --output results/kokoro-dora.json
voice-agent/.venv/bin/python scripts/voice_bench.py --corpus docs/voice-lab/corpus.json --tts pocket --voice rafael --output results/pocket-rafael.json
```

O STT calcula WER com normalização simples de pontuação e maiúsculas. Datas/números escritos de formas equivalentes exigem avaliação manual. O TTS salva WAVs de avaliação e mede primeiro PCM/tempo total. A escuta e as notas de naturalidade não são automatizadas.

Para ponta a ponta, na web:

1. Use primeiro scripted e altere só `base`/`small`.
2. Fixe STT e compare Kokoro/Pocket; depois as três vozes Kokoro.
3. Fixe voz e compare os motores reais elegíveis.
4. Compare silêncio fixo com Smart Turn usando pausas, “aham” e correções.
5. Execute 3 aquecimentos e ao menos 30 turnos por perfil; repita para obter um p95 mais representativo. Teste resposta curta, longa, interrupção e ferramenta.
6. Exporte as medições na interface e execute:

```bash
python3 scripts/voice_bench.py CAMINHO_DO_EXPORT.json
python3 scripts/voice_bench.py --template results/manual.csv
python3 scripts/monitor_lab.py --seconds 120 --output results/resources.csv
```

`monitor_lab` acompanha as árvores dos processos iniciados por `jarvis run`, RAM/swap da máquina e temperaturas quando disponíveis. PostgreSQL Docker não pertence à árvore desses três processos; meça seu consumo também via `docker stats`. CPU pode exceder 100% por usar vários núcleos.

Metas iniciais: fim físico da fala até primeiro áudio **mediana <=1,5 s / p95 <=2,5 s**; início da interrupção até silêncio **p95 <=300 ms**. As métricas incluem o atraso estimado de fim de turno, mas horários de recepção de eventos e AudioContext são aproximações. Não confundimos “mute solicitado” com silêncio acústico: essa última medida exige gravação de loopback/microfone ou instrumento externo. O resumo não inventa esse número.

Tabela manual mínima:

| Perfil | Dispositivo/rede | Precisão PT-BR | Pronúncia/naturalidade | Mediana/p95 de áudio | p95 interrupção | CPU/RAM/swap/temp. | Falsas interrupções/áudio atrasado | Reconexão/cota |
|---|---|---|---|---|---|---|---|---|
| base + Kokoro Dora + scripted | notebook/fone | pendente | pendente | pendente | pendente | pendente | pendente | pendente |
| small + Kokoro Dora + scripted | notebook/fone | pendente | pendente | pendente | pendente | pendente | pendente | pendente |
| STT fixado + Pocket Rafael + scripted | notebook/fone | pendente | pendente | pendente | pendente | pendente | pendente | pendente |
| voz fixada + motor real/modelo | notebook e Android/4G | pendente | pendente | pendente | pendente | pendente | pendente | pendente |

## Critérios para a segunda rodada

- CPU/RAM/STT: whisper.cpp base/small quantizado, threads equivalentes; OpenVINO somente se o encoder dominar.
- Parciais necessárias: Nemotron streaming 160/560 ms ou WhisperLiveKit. Verificar separadamente licença, idiomas e compatibilidade do checkpoint PT-BR com os exports; blocos não são latência ponta a ponta.
- Diagnosticar STT local: Groq Whisper Turbo somente com opt-in de envio de áudio e confirmação Free.
- TTS leve insuficiente: Supertonic 3 INT8; Piper como reserva de consumo, após revisar a voz.
- Qualidade ainda insuficiente e RAM disponível: Qwen3-TTS/Chatterbox.
- Referências externas: Gemini Live, Transcribe Live e ElevenLabs só após confirmar modelo/cota gratuitos, usando dados fictícios. Não são dependências desta entrega.

## Migração, histórico e reversão

`002_conversations.sql` e `003_delivery_status.sql` são aditivas. `schema_migrations` e advisory lock tornam a aplicação repetível; o Core só permite uma instância sobre o banco. Na subida, turnos que estavam ativos tornam-se interrompidos, recuperando deltas persistidos; ferramentas reclamadas sem resultado ficam incertas e não são repetidas.

Faça backup antes de migrar o banco real:

```bash
mkdir -p .local/backups
chmod 700 .local/backups
docker exec jarvis-postgres pg_dump -U jarvis -d jarvis > .local/backups/before-voice-lab.sql
chmod 600 .local/backups/before-voice-lab.sql
```

A reversão normal é parar o laboratório e voltar ao código/perfil anterior, preservando as tabelas novas. Não oferecemos `DROP TABLE` automático. A migração original só tinha metadados: não havia conversas persistidas a converter.

O PostgreSQL é autoridade para histórico e execuções. Python mantém buffers transitórios; o browser conserva apenas IDs de conversa/sessão. O campo `status` descreve geração; `delivery_status` descreve áudio pendente, interrompido ou finalizado de forma estimada. Assim, uma interrupção após o último token ainda fica registrada. O progresso de reprodução é estimado e limitado aos milissegundos de áudio enviados, distinguindo-o de texto gerado. Não há promessa de comprovar o que foi ouvido nem de execução exatamente uma vez para efeitos externos.

Os detalhes de eventos, controle, retomada e estados estão em [protocol.md](protocol.md).
