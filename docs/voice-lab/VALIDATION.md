# Evidências de validação — 2026-11-13

## Executado no ambiente remoto

| Verificação | Resultado |
|---|---|
| `go test -race -p 4 ./...` | Passou, incluindo testes legados e novos de streaming, quota, cancelamento e protocolo Codex simulado |
| Build do Core | Passou |
| Python `pytest -q` | 18 passaram |
| `pip check` | Sem requisitos quebrados |
| Import de faster-whisper, Kokoro e Pocket | Passou; não equivale a carregar pesos |
| Silero ONNX e Smart Turn v3.2 ONNX | Carregamento e processamento de silêncio passaram |
| `npm run lint` | Passou |
| `npm run build` | Passou, incluindo tipos e rota `/lab` |
| `scripts/jarvis setup` | Executado com sucesso na instalação existente |
| `scripts/jarvis run --synthetic ...` | Iniciou Go, Python, Next.js e PostgreSQL; reaplicação das migrações sem erro |
| Encerramento do runner por SIGINT | Retornou zero e encerrou os grupos de processos iniciados por ele |
| `core_smoke.py` | Passou: persistência, replay de eventos, idempotência de turno, ferramenta, cancelamento, progresso monotônico e origem negada |
| Interrupção após último token | Passou: `delivery_status=interrupted` preservado mesmo com geração completa e ACK final atrasado |
| `rtc_smoke.py` | Passou: cliente aiortc recebeu áudio sintético, texto, ferramenta e interrupção; reconectou e preservou conversa sem repetir ferramenta |
| Eventos obsoletos | Teste de fila descartou geração antiga; smoke verificou epoch do cliente após interrupção |
| Navegador Chromium/Playwright | UI, seleção Pocket/Rafael e Smart Turn, rota de perfil e proxy do Core passaram, sem erro de página |
| `monitor_lab.py` | Gerou CSV de exercício; não é benchmark do notebook |
| `git diff --check` | Passou |

O detector de corrida encontrou inicialmente `gin.SetMode` sendo alterado em
paralelo pelos testes legados. A configuração global foi movida para `TestMain`,
e a suíte completa passou sem desabilitar o detector.

## Bloqueado ou não executado

- Hugging Face: acesso ao config do modelo recebeu 403 do proxy. Não foram
  baixados/carregados pesos Whisper/Kokoro/Pocket para inferência real.
- Documentação específica de Sign in with ChatGPT/Groq: bloqueada por política de
  rede. Foram inspecionados schemas do CLI e repositórios oficiais acessíveis,
  mas isso não confirma enquadramento, elegibilidade ou cota da conta.
- Codex: sem login do usuário; protocolo exercitado com servidor simulado.
- Groq: sem chave/organização Free confirmada; `429` exercitado com servidor HTTP
  simulado, sem consumir artificialmente uma franquia real.
- Ollama: modelo do notebook, digest e quantização não estão neste executor.
- Mídia no Chromium: a política gerenciada
  `WebRtcIPHandling=disable_non_proxied_udp` impediu a conexão UDP. Não removemos
  essa política. O sucesso do cliente aiortc não substitui o teste no navegador.
- Notebook i5/8 GB, microfone, alto-falantes, Android e tailnet: não acessíveis.
- Licença da voz Pocket Rafael: revisão individual permanece necessária.
- Precisão, naturalidade PT-BR, cotas reais e metas 1,5/2,5 s e 300 ms: pendentes.

Nenhum teste sintético foi apresentado como inferência real ou prova de qualidade
acústica. Nenhuma cobrança, publicação ou configuração de tailnet foi ativada.
