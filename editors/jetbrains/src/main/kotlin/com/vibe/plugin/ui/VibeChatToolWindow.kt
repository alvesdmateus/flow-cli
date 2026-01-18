package com.vibe.plugin.ui

import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.SimpleToolWindowPanel
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextArea
import com.intellij.ui.components.JBTextField
import com.intellij.ui.content.ContentFactory
import com.vibe.plugin.VibeService
import java.awt.BorderLayout
import java.awt.event.KeyAdapter
import java.awt.event.KeyEvent
import javax.swing.BoxLayout
import javax.swing.JButton
import javax.swing.JPanel
import javax.swing.SwingUtilities

class VibeChatToolWindowFactory : ToolWindowFactory, DumbAware {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        val chatPanel = VibeChatPanel(project)
        val content = ContentFactory.getInstance().createContent(chatPanel, "", false)
        toolWindow.contentManager.addContent(content)
    }
}

class VibeChatPanel(private val project: Project) : SimpleToolWindowPanel(true, true) {
    private val messagesArea: JBTextArea
    private val inputField: JBTextField
    private val sendButton: JButton
    private val clearButton: JButton

    init {
        // Messages area
        messagesArea = JBTextArea().apply {
            isEditable = false
            lineWrap = true
            wrapStyleWord = true
        }

        // Input area
        inputField = JBTextField().apply {
            addKeyListener(object : KeyAdapter() {
                override fun keyPressed(e: KeyEvent) {
                    if (e.keyCode == KeyEvent.VK_ENTER && !e.isShiftDown) {
                        sendMessage()
                        e.consume()
                    }
                }
            })
        }

        sendButton = JButton("Send").apply {
            addActionListener { sendMessage() }
        }

        clearButton = JButton("Clear").apply {
            addActionListener { clearChat() }
        }

        // Layout
        val inputPanel = JPanel().apply {
            layout = BoxLayout(this, BoxLayout.X_AXIS)
            add(inputField)
            add(sendButton)
            add(clearButton)
        }

        val mainPanel = JPanel(BorderLayout()).apply {
            add(JBScrollPane(messagesArea), BorderLayout.CENTER)
            add(inputPanel, BorderLayout.SOUTH)
        }

        setContent(mainPanel)

        // Welcome message
        appendMessage("System", "Welcome to Vibe Chat! Type a message and press Enter to send.")
    }

    private fun sendMessage() {
        val message = inputField.text.trim()
        if (message.isEmpty()) return

        inputField.text = ""
        appendMessage("You", message)

        val service = VibeService.getInstance(project)
        if (!service.isRunning) {
            appendMessage("System", "Server not running. Starting...")
            service.start()
        }

        service.executeCommand("vibe.runPrompt", listOf(message)).thenAccept { result ->
            SwingUtilities.invokeLater {
                val response = (result as? Map<*, *>)?.get("message")?.toString()
                    ?: "No response received"
                appendMessage("Vibe", response)
            }
        }.exceptionally { error ->
            SwingUtilities.invokeLater {
                appendMessage("Error", error.message ?: "Unknown error")
            }
            null
        }
    }

    private fun clearChat() {
        messagesArea.text = ""
        appendMessage("System", "Chat cleared.")
    }

    private fun appendMessage(role: String, content: String) {
        val formatted = "[$role] $content\n\n"
        messagesArea.append(formatted)
        messagesArea.caretPosition = messagesArea.document.length
    }
}
