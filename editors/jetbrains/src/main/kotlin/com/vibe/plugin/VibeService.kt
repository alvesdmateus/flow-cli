package com.vibe.plugin

import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.service
import com.intellij.openapi.project.Project
import com.vibe.plugin.settings.VibeSettings
import java.io.BufferedReader
import java.io.BufferedWriter
import java.io.InputStreamReader
import java.io.OutputStreamWriter
import java.util.concurrent.CompletableFuture
import com.google.gson.Gson
import com.google.gson.JsonObject

@Service(Service.Level.PROJECT)
class VibeService(private val project: Project) {
    private var process: Process? = null
    private var writer: BufferedWriter? = null
    private var reader: BufferedReader? = null
    private val gson = Gson()
    private var messageId = 0

    companion object {
        fun getInstance(project: Project): VibeService = project.service()
    }

    val isRunning: Boolean
        get() = process?.isAlive == true

    fun start() {
        if (isRunning) {
            notify("Vibe server is already running", NotificationType.INFORMATION)
            return
        }

        val settings = VibeSettings.getInstance()
        val cmd = mutableListOf(settings.serverPath, "lsp")
        if (settings.model.isNotEmpty()) {
            cmd.add("--model")
            cmd.add(settings.model)
        }

        try {
            val builder = ProcessBuilder(cmd)
            builder.redirectErrorStream(true)
            process = builder.start()
            writer = BufferedWriter(OutputStreamWriter(process!!.outputStream))
            reader = BufferedReader(InputStreamReader(process!!.inputStream))

            // Send initialize request
            initialize()

            notify("Vibe server started", NotificationType.INFORMATION)
        } catch (e: Exception) {
            notify("Failed to start Vibe server: ${e.message}", NotificationType.ERROR)
        }
    }

    fun stop() {
        if (!isRunning) {
            notify("Vibe server is not running", NotificationType.INFORMATION)
            return
        }

        try {
            // Send shutdown request
            sendRequest("shutdown", null)
            sendNotification("exit", null)

            process?.destroy()
            process = null
            writer = null
            reader = null

            notify("Vibe server stopped", NotificationType.INFORMATION)
        } catch (e: Exception) {
            notify("Error stopping Vibe server: ${e.message}", NotificationType.WARNING)
            process?.destroyForcibly()
        }
    }

    private fun initialize() {
        val params = JsonObject().apply {
            addProperty("processId", ProcessHandle.current().pid().toInt())
            addProperty("rootUri", project.basePath?.let { "file://$it" })
            add("capabilities", JsonObject())
        }
        sendRequest("initialize", params)
        sendNotification("initialized", JsonObject())
    }

    fun executeCommand(command: String, arguments: List<Any>): CompletableFuture<Any?> {
        if (!isRunning) {
            return CompletableFuture.failedFuture(IllegalStateException("Server not running"))
        }

        val params = JsonObject().apply {
            addProperty("command", command)
            add("arguments", gson.toJsonTree(arguments))
        }

        return sendRequestAsync("workspace/executeCommand", params)
    }

    private fun sendRequest(method: String, params: JsonObject?): JsonObject? {
        val id = ++messageId
        val message = JsonObject().apply {
            addProperty("jsonrpc", "2.0")
            addProperty("id", id)
            addProperty("method", method)
            params?.let { add("params", it) }
        }

        sendMessage(message)
        return readResponse()
    }

    private fun sendRequestAsync(method: String, params: JsonObject?): CompletableFuture<Any?> {
        return CompletableFuture.supplyAsync {
            val response = sendRequest(method, params)
            response?.get("result")
        }
    }

    private fun sendNotification(method: String, params: JsonObject?) {
        val message = JsonObject().apply {
            addProperty("jsonrpc", "2.0")
            addProperty("method", method)
            params?.let { add("params", it) }
        }
        sendMessage(message)
    }

    private fun sendMessage(message: JsonObject) {
        val content = gson.toJson(message)
        val header = "Content-Length: ${content.toByteArray().size}\r\n\r\n"

        writer?.apply {
            write(header)
            write(content)
            flush()
        }
    }

    private fun readResponse(): JsonObject? {
        val headerLine = reader?.readLine() ?: return null
        if (!headerLine.startsWith("Content-Length:")) return null

        val contentLength = headerLine.substringAfter(":").trim().toInt()
        reader?.readLine() // Empty line

        val content = CharArray(contentLength)
        reader?.read(content, 0, contentLength)

        return gson.fromJson(String(content), JsonObject::class.java)
    }

    private fun notify(message: String, type: NotificationType) {
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Vibe Notifications")
            .createNotification(message, type)
            .notify(project)
    }
}
