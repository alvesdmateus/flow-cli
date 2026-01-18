package com.vibe.plugin.lsp

import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.platform.lsp.api.LspServerSupportProvider
import com.intellij.platform.lsp.api.ProjectWideLspServerDescriptor
import com.vibe.plugin.settings.VibeSettings

/**
 * LSP Server Support Provider for Vibe.
 * This integrates vibe-cli's LSP server with JetBrains IDE's LSP support.
 */
class VibeLspServerSupportProvider : LspServerSupportProvider {
    override fun fileOpened(
        project: Project,
        file: VirtualFile,
        serverStarter: LspServerSupportProvider.LspServerStarter
    ) {
        val extension = file.extension?.lowercase() ?: return

        val supportedExtensions = listOf(
            "go", "py", "js", "ts", "jsx", "tsx",
            "rs", "java", "c", "cpp", "h", "hpp",
            "cs", "rb", "php", "swift", "kt", "kts"
        )

        if (extension in supportedExtensions) {
            serverStarter.ensureServerStarted(VibeLspServerDescriptor(project))
        }
    }
}

class VibeLspServerDescriptor(project: Project) : ProjectWideLspServerDescriptor(project, "Vibe") {
    override fun isSupportedFile(file: VirtualFile): Boolean {
        val extension = file.extension?.lowercase() ?: return false
        return extension in listOf(
            "go", "py", "js", "ts", "jsx", "tsx",
            "rs", "java", "c", "cpp", "h", "hpp",
            "cs", "rb", "php", "swift", "kt", "kts"
        )
    }

    override fun createCommandLine(): com.intellij.execution.configurations.GeneralCommandLine {
        val settings = VibeSettings.getInstance()
        val cmd = com.intellij.execution.configurations.GeneralCommandLine(settings.serverPath)
        cmd.addParameter("lsp")

        if (settings.model.isNotEmpty()) {
            cmd.addParameter("--model")
            cmd.addParameter(settings.model)
        }

        return cmd
    }
}
