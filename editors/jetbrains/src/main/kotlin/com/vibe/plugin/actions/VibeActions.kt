package com.vibe.plugin.actions

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.wm.ToolWindowManager
import com.vibe.plugin.VibeService

abstract class BaseVibeAction : AnAction() {
    protected fun getSelection(editor: Editor): String? {
        val selectionModel = editor.selectionModel
        return if (selectionModel.hasSelection()) {
            selectionModel.selectedText
        } else {
            null
        }
    }

    protected fun showResult(project: Project, title: String, message: String) {
        Messages.showInfoMessage(project, message, title)
    }

    protected fun ensureServerRunning(project: Project): Boolean {
        val service = VibeService.getInstance(project)
        if (!service.isRunning) {
            val result = Messages.showYesNoDialog(
                project,
                "Vibe server is not running. Start it now?",
                "Vibe",
                Messages.getQuestionIcon()
            )
            if (result == Messages.YES) {
                service.start()
                return service.isRunning
            }
            return false
        }
        return true
    }
}

class ExplainCodeAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val editor = e.getData(CommonDataKeys.EDITOR) ?: return

        val selection = getSelection(editor)
        if (selection.isNullOrEmpty()) {
            Messages.showWarningDialog(project, "Please select code to explain", "Vibe")
            return
        }

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.explainCode", listOf(selection)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No explanation available"
            showResult(project, "Code Explanation", message)
        }
    }

    override fun update(e: AnActionEvent) {
        val editor = e.getData(CommonDataKeys.EDITOR)
        e.presentation.isEnabled = editor?.selectionModel?.hasSelection() == true
    }
}

class GenerateTestsAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val editor = e.getData(CommonDataKeys.EDITOR) ?: return

        val selection = getSelection(editor) ?: editor.document.text

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.generateTests", listOf(selection)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No tests generated"
            showResult(project, "Generated Tests", message)
        }
    }
}

class RefactorAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val editor = e.getData(CommonDataKeys.EDITOR) ?: return

        val selection = getSelection(editor)
        if (selection.isNullOrEmpty()) {
            Messages.showWarningDialog(project, "Please select code to refactor", "Vibe")
            return
        }

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.refactor", listOf(selection)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No suggestions available"
            showResult(project, "Refactoring Suggestions", message)
        }
    }

    override fun update(e: AnActionEvent) {
        val editor = e.getData(CommonDataKeys.EDITOR)
        e.presentation.isEnabled = editor?.selectionModel?.hasSelection() == true
    }
}

class FixErrorAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val editor = e.getData(CommonDataKeys.EDITOR) ?: return

        // Get error at cursor position (simplified - in real impl would use HighlightInfo)
        val caretOffset = editor.caretModel.offset
        val document = editor.document
        val lineNumber = document.getLineNumber(caretOffset)

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.fixError", listOf(lineNumber)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No fix available"
            showResult(project, "Fix Suggestion", message)
        }
    }
}

class GenerateDocsAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val editor = e.getData(CommonDataKeys.EDITOR) ?: return

        val selection = getSelection(editor) ?: editor.document.text

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.generateDocs", listOf(selection)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No documentation generated"
            showResult(project, "Generated Documentation", message)
        }
    }
}

class OpenChatAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val toolWindow = ToolWindowManager.getInstance(project).getToolWindow("Vibe Chat")
        toolWindow?.show()
    }
}

class RunPromptAction : BaseVibeAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return

        val prompt = Messages.showInputDialog(
            project,
            "Enter your prompt:",
            "Vibe Prompt",
            Messages.getQuestionIcon()
        )

        if (prompt.isNullOrEmpty()) return

        if (!ensureServerRunning(project)) return

        val service = VibeService.getInstance(project)
        service.executeCommand("vibe.runPrompt", listOf(prompt)).thenAccept { result ->
            val message = (result as? Map<*, *>)?.get("message")?.toString() ?: "No response"
            showResult(project, "Vibe Response", message)
        }
    }
}
