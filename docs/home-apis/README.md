# Prova Android das Google Home APIs

Estado em 2026-11-15: **probe Jarvis somente leitura implementado e compilado**.
OAuth e teste em Android físico ainda pendentes.

## Objetivo desta prova

Usar o sample oficial Android para verificar, primeiro em modo leitura, se a
lâmpada “Luz Do Quarto” (serviço NovaDigital Smart; nome de origem “luzinha”)
aparece na casa Google e que capacidades a API informa. Não fazer pareamento,
reset, desvinculação nem enviar comandos nesta etapa.

## Requisitos oficiais conferidos

- Google Home APIs Android estão em open beta. O SDK não é distribuído como
  dependência pública padrão: a documentação pede entrar no Google Home
  Developers para baixar e hospedar as bibliotecas localmente.
- A documentação lista Android Studio 2024.2.1 (“Ladybug”) ou superior e ADB
  para desenvolvimento; Android 10+, conta Google de desenvolvimento, dispositivo
  físico e Wi-Fi para testar. Recomenda aparelho físico por depender do Google
  Play Services.
- Para teste, registro/aprovação no Google Home Developer Console não é exigido,
  mas OAuth precisa ser configurado e a conta do usuário adicionada como test
  user. App não verificado tem limite documentado de 100 test users. O Developer
  Console de registro Home ainda aparece como indisponível na documentação.
- Consentimento dá acesso à estrutura selecionada, não substitui a allowlist
  própria do Jarvis. A permissão pode ser revogada no Google Home.
- A página do sample documenta um escopo padrão de dispositivos selecionados e
  alerta que o escopo `home.platform.v2` depende de aprovação de parceiro para
  alguns recursos. Não solicitar escopo avançado para descobrir uma lâmpada.

Fontes oficiais:

- [Get started](https://developers.home.google.com/apis/android/get-started)
- [SDK](https://developers.home.google.com/apis/android/sdk)
- [Sample App/build](https://developers.home.google.com/apis/android/sample-app/build)
- [OAuth](https://developers.home.google.com/apis/android/oauth)
- [Permissions](https://developers.home.google.com/apis/android/permissions)
- [Supported device types](https://developers.home.google.com/apis/android/supported-device-types)
- [Control devices](https://developers.home.google.com/apis/android/device/control)

## Investigação local executada

- Clonado fora do repositório em `/tmp/opencode/google-home-api-sample-app-android`
  o sample oficial `google-home/google-home-api-sample-app-android`, commit
  `9d5405bb9b086cafe9e735fecbaaeaf053647f75` (2026-11-03, sample 1.11.1).
- O sample usa Gradle Wrapper 9.2.1, Android Gradle Plugin 8.13.2, Kotlin 2.4.0,
  compileSdk 36/minSdk 29 e declara Home APIs 17.1.0.
- O usuário extraiu o SDK em `/home/jean/Projetos/com`. Confirmado que contém
  apenas as duas bibliotecas Home APIs 17.1.0 e seus POMs/checksums/javadocs.
  Não havia `~/.m2/repository/com`; os artefatos foram copiados sem conflito para
  `~/.m2/repository/com/google/android/gms`, preservando a pasta de origem.
- O Gradle Wrapper executou por linha de comando e instalou Android SDK Platform
  36 e Build Tools 35 no SDK local existente. Com as bibliotecas em Maven local,
  o sample oficial compilou após instalar JDK 17. Foi criado `android/home-probe`,
  package `com.jarvis.homeprobe`, minSdk 29, Home APIs scope v1, registro limitado
  a quatro tipos de luz e sem botão/método de comando. O probe exibe apenas nome e
  cômodo; não lê estado nem envia operações. Build CLI `:app:assembleDebug` passou.
- APK debug em `android/home-probe/app/build/outputs/apk/debug/app-debug.apk`
  (~33 MB, ignorado pelo Git), assinado pelo debug keystore local. Package e
  fingerprint foram conferidos com `aapt`/`apksigner`; não registrar fingerprint
  específico do mantenedor no repositório.
- Não há Gradle global; usar wrapper. ADB existe, mas `adb devices` está vazio.
  Nenhuma conta Google foi autorizada e nenhum dispositivo foi consultado.
- Não copiar o keystore de debug do sample oficial nem reutilizar identidade,
  package name ou OAuth client do sample. Jarvis precisa de identidade e
  fingerprint de debug próprios antes de configurar OAuth.
- A cópia local `google-content.html` registra o download do SDK do bucket
  `home_sdk_android` e orienta extrair `com` para `~/.m2/repository`.
- O HTML contém metadados temporários de sessão: não compartilhar, enviar a
  terceiros ou versionar; preferir removê-lo/movê-lo para fora do repo após
  extrair instruções não sensíveis.

## Próximo passo (requer o mantenedor)

1. Criar/selecionar projeto no Google Cloud (não habilitar billing), configurar
   OAuth como app de teste e adicionar a própria conta Google como test user.
2. Criar OAuth client tipo Android com package `com.jarvis.homeprobe` e SHA-1
   mostrado localmente por `apksigner`. Não enviar credenciais ou chave; a
   impressão digital não é senha, mas vincula este build à credencial.
3. Usar a mesma conta de teste em Android 10+ com Google Play Services e Wi-Fi.
   Instalar o APK via ADB após autorizar depuração USB ou transferir/instalar
   manualmente; revisar o consentimento antes de aceitar.
4. Apps não verificados podem receber acesso amplo à estrutura conforme tipos
   elegíveis. O probe só registra quatro tipos de luz e mostra nome/cômodo, sem
   comandos; se o consentimento indicar escopo inesperado, cancelar e consultar.
5. Descobrir luzes sem enviar comandos. Se a lâmpada não aparecer, conferir conta,
   estrutura e autorização antes de concluir incompatibilidade.

Se o acesso ao SDK exigir aprovação/parceria indisponível, ou OAuth negar
permissões, parar e reavaliar a estratégia antes de criar a ponte Jarvis. A
presença do sample/capability `OnOff` não prova que o dispositivo NovaDigital
específico seja exposto.

## Segurança e etapa futura

- A conta Google e tokens permanecem no aparelho; nunca no backend, modelo,
  logs, screenshots ou Git.
- Discovery inicial somente leitura. O usuário confirma qual dispositivo
  corresponde à lâmpada; apelido não é identidade estável.
- Controle físico exige etapa separada e autorização explícita. Usar `on()` e
  `off()` apenas se o trait expuser esses comandos; não usar `toggle()` nem repetir
  automaticamente comandos após timeout/reconexão.
- A futura ponte Android ↔ Core precisa de protocolo autenticado, canal privado,
  validade/deduplicação e journal com resultado enviado/confirmado/incerto.
