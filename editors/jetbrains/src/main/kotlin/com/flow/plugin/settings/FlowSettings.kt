package com.flow.plugin.settings

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

@State(
    name = "FlowSettings",
    storages = [Storage("vibe.xml")]
)
class FlowSettings : PersistentStateComponent<FlowSettings.State> {
    private var myState = State()

    companion object {
        fun getInstance(): FlowSettings =
            ApplicationManager.getApplication().getService(FlowSettings::class.java)
    }

    data class State(
        var serverPath: String = "flow",
        var model: String = "",
        var provider: String = "ollama",
        var ollamaUrl: String = "http://localhost:11434",
        var autoStart: Boolean = true,
        var logLevel: String = "info"
    )

    var serverPath: String
        get() = myState.serverPath
        set(value) { myState.serverPath = value }

    var model: String
        get() = myState.model
        set(value) { myState.model = value }

    var provider: String
        get() = myState.provider
        set(value) { myState.provider = value }

    var ollamaUrl: String
        get() = myState.ollamaUrl
        set(value) { myState.ollamaUrl = value }

    var autoStart: Boolean
        get() = myState.autoStart
        set(value) { myState.autoStart = value }

    var logLevel: String
        get() = myState.logLevel
        set(value) { myState.logLevel = value }

    override fun getState(): State = myState

    override fun loadState(state: State) {
        myState = state
    }
}
