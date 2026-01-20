package com.flow.plugin.settings

import com.intellij.openapi.options.Configurable
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import javax.swing.JComboBox
import javax.swing.JComponent
import javax.swing.JPanel

class FlowSettingsConfigurable : Configurable {
    private var settingsPanel: JPanel? = null
    private var serverPathField: JBTextField? = null
    private var modelField: JBTextField? = null
    private var providerCombo: JComboBox<String>? = null
    private var ollamaUrlField: JBTextField? = null
    private var autoStartCheckbox: JBCheckBox? = null
    private var logLevelCombo: JComboBox<String>? = null

    override fun getDisplayName(): String = "Flow"

    override fun createComponent(): JComponent {
        serverPathField = JBTextField()
        modelField = JBTextField()
        providerCombo = JComboBox(arrayOf("ollama", "openai-compatible", "localai", "lmstudio", "vllm"))
        ollamaUrlField = JBTextField()
        autoStartCheckbox = JBCheckBox("Auto-start server when project opens")
        logLevelCombo = JComboBox(arrayOf("error", "warn", "info", "debug"))

        settingsPanel = FormBuilder.createFormBuilder()
            .addLabeledComponent(JBLabel("Server path:"), serverPathField!!, 1, false)
            .addLabeledComponent(JBLabel("Model:"), modelField!!, 1, false)
            .addLabeledComponent(JBLabel("Provider:"), providerCombo!!, 1, false)
            .addLabeledComponent(JBLabel("Ollama URL:"), ollamaUrlField!!, 1, false)
            .addComponent(autoStartCheckbox!!, 1)
            .addLabeledComponent(JBLabel("Log level:"), logLevelCombo!!, 1, false)
            .addComponentFillVertically(JPanel(), 0)
            .panel

        return settingsPanel!!
    }

    override fun isModified(): Boolean {
        val settings = FlowSettings.getInstance()
        return serverPathField?.text != settings.serverPath ||
                modelField?.text != settings.model ||
                providerCombo?.selectedItem != settings.provider ||
                ollamaUrlField?.text != settings.ollamaUrl ||
                autoStartCheckbox?.isSelected != settings.autoStart ||
                logLevelCombo?.selectedItem != settings.logLevel
    }

    override fun apply() {
        val settings = FlowSettings.getInstance()
        settings.serverPath = serverPathField?.text ?: "flow"
        settings.model = modelField?.text ?: ""
        settings.provider = providerCombo?.selectedItem as? String ?: "ollama"
        settings.ollamaUrl = ollamaUrlField?.text ?: "http://localhost:11434"
        settings.autoStart = autoStartCheckbox?.isSelected ?: true
        settings.logLevel = logLevelCombo?.selectedItem as? String ?: "info"
    }

    override fun reset() {
        val settings = FlowSettings.getInstance()
        serverPathField?.text = settings.serverPath
        modelField?.text = settings.model
        providerCombo?.selectedItem = settings.provider
        ollamaUrlField?.text = settings.ollamaUrl
        autoStartCheckbox?.isSelected = settings.autoStart
        logLevelCombo?.selectedItem = settings.logLevel
    }
}
