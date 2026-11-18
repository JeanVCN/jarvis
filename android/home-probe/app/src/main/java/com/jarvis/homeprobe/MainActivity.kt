package com.jarvis.homeprobe

import android.os.Bundle
import android.view.ViewGroup
import android.widget.Button
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.activity.ComponentActivity
import androidx.lifecycle.lifecycleScope
import com.google.home.FactoryRegistry
import com.google.home.Home
import com.google.home.HomeClient
import com.google.home.HomeConfig
import com.google.home.PermissionsResultStatus
import com.google.home.PermissionsState
import com.google.home.TraitFactory
import com.google.home.DeviceTypeFactory
import com.google.home.DeviceType
import com.google.home.Trait
import com.google.home.matter.standard.BasicInformation
import com.google.home.matter.standard.ColorTemperatureLightDevice
import com.google.home.matter.standard.DimmableLightDevice
import com.google.home.matter.standard.ExtendedColorLightDevice
import com.google.home.matter.standard.OnOff
import com.google.home.matter.standard.OnOffLightDevice
import com.google.home.matter.standard.LevelControl
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch

/**
 * Temporary read-only compatibility probe. It deliberately has no device
 * command controls and registers only light device types.
 */
class MainActivity : ComponentActivity() {
    private lateinit var homeClient: HomeClient
    private lateinit var statusView: TextView
    private lateinit var inventoryView: TextView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val lightTypes: List<DeviceTypeFactory<out DeviceType>> = listOf(
            OnOffLightDevice,
            DimmableLightDevice,
            ColorTemperatureLightDevice,
            ExtendedColorLightDevice,
        )
        val lightTraits: List<TraitFactory<out Trait>> = listOf(
            BasicInformation,
            OnOff,
            LevelControl,
        )
        val config = HomeConfig(
            coroutineContext = kotlinx.coroutines.Dispatchers.IO,
            factoryRegistry = FactoryRegistry(types = lightTypes, traits = lightTraits),
            homePlatformScope = HomeConfig.HomePlatformScope.HOME_PLATFORM_SCOPE_VERSION_1,
        )
        homeClient = Home.getClient(applicationContext, homeConfig = config)
        homeClient.registerActivityResultCallerForPermissions(this)

        val title = TextView(this).apply {
            text = "Jarvis — diagnóstico Google Home"
            textSize = 22f
        }
        statusView = TextView(this).apply { textSize = 16f }
        inventoryView = TextView(this).apply { textSize = 16f }
        val consentButton = Button(this).apply {
            text = "Autorizar acesso para diagnóstico"
            setOnClickListener { requestReadAccess() }
        }
        val refreshButton = Button(this).apply {
            text = "Atualizar lista de luzes"
            setOnClickListener { discoverLights() }
        }
        val explanation = TextView(this).apply {
            text = "Probe temporário somente leitura. Não oferece comandos para ligar, desligar ou alterar dispositivos."
            textSize = 14f
        }

        val content = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(24, 24, 24, 24)
            addView(title, matchWidth())
            addView(explanation, spaced(matchWidth(), 12))
            addView(statusView, spaced(matchWidth(), 20))
            addView(consentButton, spaced(matchWidth(), 12))
            addView(refreshButton, spaced(matchWidth(), 8))
            addView(inventoryView, spaced(matchWidth(), 20))
        }
        setContentView(ScrollView(this).apply { addView(content) })

        lifecycleScope.launch {
            when (homeClient.hasPermissions().first()) {
                PermissionsState.GRANTED -> {
                    statusView.text = "Acesso de leitura concedido."
                    discoverLights()
                }
                else -> statusView.text = "Aguardando autorização explícita."
            }
        }
    }

    private fun requestReadAccess() {
        lifecycleScope.launch {
            statusView.text = "Abrindo consentimento do Google…"
            try {
                val result = homeClient.requestPermissions()
                statusView.text = when (result.status) {
                    PermissionsResultStatus.SUCCESS -> "Acesso concedido. Lendo dispositivos…"
                    PermissionsResultStatus.CANCELLED -> "Autorização cancelada; nenhum dispositivo foi consultado."
                    PermissionsResultStatus.ERROR -> "Falha na autorização: ${result.errorMessage ?: "erro não informado"}"
                }
                if (result.status == PermissionsResultStatus.SUCCESS) discoverLights()
            } catch (error: Exception) {
                statusView.text = "Não foi possível solicitar acesso: ${error.message ?: error.javaClass.simpleName}"
            }
        }
    }

    private fun discoverLights() {
        lifecycleScope.launch {
            try {
                if (homeClient.hasPermissions().first() != PermissionsState.GRANTED) {
                    statusView.text = "Autorize primeiro o acesso de leitura."
                    return@launch
                }
                statusView.text = "Consultando tipos de luz na casa autorizada…"
                val lines = mutableListOf<String>()
                for (structure in homeClient.structures().first()) {
                    val rooms = structure.rooms().first().associate { it.id.id to it.name }
                    for (device in structure.devices().first()) {
                        val isLight = device.has(OnOffLightDevice) ||
                            device.has(DimmableLightDevice) ||
                            device.has(ColorTemperatureLightDevice) ||
                            device.has(ExtendedColorLightDevice)
                        if (isLight) {
                            val roomName = device.roomId?.id?.let(rooms::get)
                            val location = roomName?.let { " — $it" } ?: ""
                            val capabilities = buildList {
                                if (device.has(OnOff)) add("liga/desliga")
                                if (device.has(LevelControl)) add("intensidade")
                            }.ifEmpty { listOf("nenhuma capacidade consultada") }
                            lines += "• ${device.name}$location\n  Capacidades: ${capabilities.joinToString(", ")}"
                        }
                    }
                }
                inventoryView.text = if (lines.isEmpty()) {
                    "Nenhuma luz dos tipos suportados foi exposta nesta casa. Isso ainda não prova incompatibilidade; registraremos os tipos/capacidades disponíveis no próximo diagnóstico."
                } else {
                    "Luzes expostas (${lines.size}):\n\n${lines.joinToString("\n")}"
                }
                statusView.text = "Leitura concluída. Nenhum comando foi enviado."
            } catch (error: Exception) {
                statusView.text = "Falha ao consultar dispositivos: ${error.message ?: error.javaClass.simpleName}"
            }
        }
    }

    private fun matchWidth() = LinearLayout.LayoutParams(
        ViewGroup.LayoutParams.MATCH_PARENT,
        ViewGroup.LayoutParams.WRAP_CONTENT,
    )

    private fun spaced(params: LinearLayout.LayoutParams, top: Int) = params.apply {
        topMargin = (top * resources.displayMetrics.density).toInt()
    }
}
