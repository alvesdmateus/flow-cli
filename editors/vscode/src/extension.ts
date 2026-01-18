import * as vscode from 'vscode';
import * as path from 'path';
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
    TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient | undefined;
let outputChannel: vscode.OutputChannel;
let chatViewProvider: VibeChatViewProvider;

export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel('Vibe');
    outputChannel.appendLine('Vibe extension activating...');

    // Initialize chat view provider
    chatViewProvider = new VibeChatViewProvider(context.extensionUri);
    context.subscriptions.push(
        vscode.window.registerWebviewViewProvider('vibeChat', chatViewProvider)
    );

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('vibe.startServer', startServer),
        vscode.commands.registerCommand('vibe.stopServer', stopServer),
        vscode.commands.registerCommand('vibe.explainCode', explainCode),
        vscode.commands.registerCommand('vibe.generateTests', generateTests),
        vscode.commands.registerCommand('vibe.refactor', refactorCode),
        vscode.commands.registerCommand('vibe.fixError', fixError),
        vscode.commands.registerCommand('vibe.generateDocs', generateDocs),
        vscode.commands.registerCommand('vibe.chat', openChat),
        vscode.commands.registerCommand('vibe.runPrompt', runPrompt)
    );

    // Auto-start if configured
    const config = vscode.workspace.getConfiguration('vibe');
    if (config.get<boolean>('autoStart', true)) {
        startServer();
    }

    outputChannel.appendLine('Vibe extension activated');
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}

async function startServer(): Promise<void> {
    if (client) {
        vscode.window.showInformationMessage('Vibe server is already running');
        return;
    }

    const config = vscode.workspace.getConfiguration('vibe');
    const serverPath = config.get<string>('serverPath', 'vibe');
    const model = config.get<string>('model', '');

    const args = ['lsp'];
    if (model) {
        args.push('--model', model);
    }

    const serverOptions: ServerOptions = {
        run: {
            command: serverPath,
            args: args,
            transport: TransportKind.stdio
        },
        debug: {
            command: serverPath,
            args: [...args, '--debug'],
            transport: TransportKind.stdio
        }
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [
            { scheme: 'file', language: 'go' },
            { scheme: 'file', language: 'python' },
            { scheme: 'file', language: 'javascript' },
            { scheme: 'file', language: 'typescript' },
            { scheme: 'file', language: 'rust' },
            { scheme: 'file', language: 'java' },
            { scheme: 'file', language: 'c' },
            { scheme: 'file', language: 'cpp' },
            { scheme: 'file', language: 'csharp' },
            { scheme: 'file', language: 'ruby' },
            { scheme: 'file', language: 'php' },
            { scheme: 'file', language: 'swift' },
            { scheme: 'file', language: 'kotlin' }
        ],
        outputChannel: outputChannel,
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.*')
        }
    };

    client = new LanguageClient(
        'vibe',
        'Vibe Language Server',
        serverOptions,
        clientOptions
    );

    try {
        await client.start();
        outputChannel.appendLine('Vibe language server started');
        vscode.window.showInformationMessage('Vibe language server started');
    } catch (error) {
        outputChannel.appendLine(`Failed to start server: ${error}`);
        vscode.window.showErrorMessage(`Failed to start Vibe server: ${error}`);
        client = undefined;
    }
}

async function stopServer(): Promise<void> {
    if (!client) {
        vscode.window.showInformationMessage('Vibe server is not running');
        return;
    }

    await client.stop();
    client = undefined;
    outputChannel.appendLine('Vibe language server stopped');
    vscode.window.showInformationMessage('Vibe language server stopped');
}

async function explainCode(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    const selection = editor.selection;
    if (selection.isEmpty) {
        vscode.window.showWarningMessage('Please select code to explain');
        return;
    }

    const selectedText = editor.document.getText(selection);
    await executeVibeCommand('vibe.explainCode', editor.document.uri.toString(), {
        start: { line: selection.start.line, character: selection.start.character },
        end: { line: selection.end.line, character: selection.end.character }
    }, selectedText);
}

async function generateTests(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    const selection = editor.selection;
    const selectedText = selection.isEmpty
        ? editor.document.getText()
        : editor.document.getText(selection);

    await executeVibeCommand('vibe.generateTests', editor.document.uri.toString(), {
        start: { line: selection.start.line, character: selection.start.character },
        end: { line: selection.end.line, character: selection.end.character }
    }, selectedText);
}

async function refactorCode(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    const selection = editor.selection;
    if (selection.isEmpty) {
        vscode.window.showWarningMessage('Please select code to refactor');
        return;
    }

    const selectedText = editor.document.getText(selection);
    await executeVibeCommand('vibe.refactor', editor.document.uri.toString(), {
        start: { line: selection.start.line, character: selection.start.character },
        end: { line: selection.end.line, character: selection.end.character }
    }, selectedText);
}

async function fixError(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    // Get diagnostics for current file
    const diagnostics = vscode.languages.getDiagnostics(editor.document.uri);
    if (diagnostics.length === 0) {
        vscode.window.showInformationMessage('No errors found in current file');
        return;
    }

    // Find error at cursor or first error
    const cursorPos = editor.selection.active;
    let targetDiagnostic = diagnostics.find(d => d.range.contains(cursorPos));
    if (!targetDiagnostic) {
        targetDiagnostic = diagnostics[0];
    }

    await executeVibeCommand('vibe.fixError', editor.document.uri.toString(), {
        start: { line: targetDiagnostic.range.start.line, character: targetDiagnostic.range.start.character },
        end: { line: targetDiagnostic.range.end.line, character: targetDiagnostic.range.end.character }
    }, targetDiagnostic.message);
}

async function generateDocs(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showWarningMessage('No active editor');
        return;
    }

    const selection = editor.selection;
    const selectedText = selection.isEmpty
        ? editor.document.getText()
        : editor.document.getText(selection);

    await executeVibeCommand('vibe.generateDocs', editor.document.uri.toString(), {
        start: { line: selection.start.line, character: selection.start.character },
        end: { line: selection.end.line, character: selection.end.character }
    }, selectedText);
}

async function openChat(): Promise<void> {
    vscode.commands.executeCommand('vibeChat.focus');
}

async function runPrompt(): Promise<void> {
    const prompt = await vscode.window.showInputBox({
        prompt: 'Enter your prompt for Vibe',
        placeHolder: 'e.g., Optimize this function for performance'
    });

    if (!prompt) {
        return;
    }

    const editor = vscode.window.activeTextEditor;
    const context = editor ? {
        uri: editor.document.uri.toString(),
        selection: editor.selection.isEmpty ? null : editor.document.getText(editor.selection),
        language: editor.document.languageId
    } : null;

    await executeVibeCommand('vibe.runPrompt', prompt, context);
}

async function executeVibeCommand(command: string, ...args: any[]): Promise<void> {
    if (!client) {
        const start = await vscode.window.showWarningMessage(
            'Vibe server is not running. Start it now?',
            'Yes', 'No'
        );
        if (start === 'Yes') {
            await startServer();
        } else {
            return;
        }
    }

    try {
        const result = await client!.sendRequest('workspace/executeCommand', {
            command: command,
            arguments: args
        });

        // Display result in chat or output
        if (result && typeof result === 'object' && 'message' in result) {
            chatViewProvider.addMessage('assistant', (result as any).message);
            vscode.commands.executeCommand('vibeChat.focus');
        }
    } catch (error) {
        outputChannel.appendLine(`Command execution failed: ${error}`);
        vscode.window.showErrorMessage(`Vibe command failed: ${error}`);
    }
}

class VibeChatViewProvider implements vscode.WebviewViewProvider {
    private _view?: vscode.WebviewView;
    private _messages: Array<{ role: string; content: string }> = [];

    constructor(private readonly _extensionUri: vscode.Uri) {}

    resolveWebviewView(
        webviewView: vscode.WebviewView,
        context: vscode.WebviewViewResolveContext,
        _token: vscode.CancellationToken
    ): void {
        this._view = webviewView;

        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [this._extensionUri]
        };

        webviewView.webview.html = this._getHtmlForWebview(webviewView.webview);

        webviewView.webview.onDidReceiveMessage(async (data) => {
            switch (data.type) {
                case 'sendMessage':
                    this.addMessage('user', data.message);
                    await this._sendToVibe(data.message);
                    break;
                case 'clear':
                    this._messages = [];
                    this._updateWebview();
                    break;
            }
        });
    }

    addMessage(role: string, content: string): void {
        this._messages.push({ role, content });
        this._updateWebview();
    }

    private async _sendToVibe(message: string): Promise<void> {
        if (!client) {
            this.addMessage('system', 'Vibe server is not running. Please start it first.');
            return;
        }

        try {
            const result = await client.sendRequest('workspace/executeCommand', {
                command: 'vibe.runPrompt',
                arguments: [message]
            });

            if (result && typeof result === 'object' && 'message' in result) {
                this.addMessage('assistant', (result as any).message);
            }
        } catch (error) {
            this.addMessage('system', `Error: ${error}`);
        }
    }

    private _updateWebview(): void {
        if (this._view) {
            this._view.webview.postMessage({
                type: 'updateMessages',
                messages: this._messages
            });
        }
    }

    private _getHtmlForWebview(webview: vscode.Webview): string {
        return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Vibe Chat</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            font-size: var(--vscode-font-size);
            color: var(--vscode-foreground);
            background: var(--vscode-editor-background);
            margin: 0;
            padding: 10px;
            display: flex;
            flex-direction: column;
            height: 100vh;
            box-sizing: border-box;
        }
        #messages {
            flex: 1;
            overflow-y: auto;
            margin-bottom: 10px;
        }
        .message {
            margin-bottom: 10px;
            padding: 8px 12px;
            border-radius: 6px;
        }
        .message.user {
            background: var(--vscode-input-background);
            margin-left: 20px;
        }
        .message.assistant {
            background: var(--vscode-editor-selectionBackground);
            margin-right: 20px;
        }
        .message.system {
            background: var(--vscode-editorWarning-foreground);
            opacity: 0.7;
            font-style: italic;
        }
        .role {
            font-weight: bold;
            margin-bottom: 4px;
            font-size: 0.85em;
            opacity: 0.8;
        }
        #input-area {
            display: flex;
            gap: 8px;
        }
        #input {
            flex: 1;
            padding: 8px;
            border: 1px solid var(--vscode-input-border);
            background: var(--vscode-input-background);
            color: var(--vscode-input-foreground);
            border-radius: 4px;
            font-family: inherit;
            font-size: inherit;
        }
        #input:focus {
            outline: none;
            border-color: var(--vscode-focusBorder);
        }
        button {
            padding: 8px 16px;
            background: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background: var(--vscode-button-hoverBackground);
        }
        #clear {
            background: var(--vscode-button-secondaryBackground);
            color: var(--vscode-button-secondaryForeground);
        }
    </style>
</head>
<body>
    <div id="messages"></div>
    <div id="input-area">
        <input type="text" id="input" placeholder="Ask Vibe anything..." />
        <button id="send">Send</button>
        <button id="clear">Clear</button>
    </div>
    <script>
        const vscode = acquireVsCodeApi();
        const messagesEl = document.getElementById('messages');
        const inputEl = document.getElementById('input');
        const sendBtn = document.getElementById('send');
        const clearBtn = document.getElementById('clear');

        function renderMessages(messages) {
            messagesEl.innerHTML = messages.map(m => \`
                <div class="message \${m.role}">
                    <div class="role">\${m.role.charAt(0).toUpperCase() + m.role.slice(1)}</div>
                    <div class="content">\${escapeHtml(m.content)}</div>
                </div>
            \`).join('');
            messagesEl.scrollTop = messagesEl.scrollHeight;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function sendMessage() {
            const message = inputEl.value.trim();
            if (message) {
                vscode.postMessage({ type: 'sendMessage', message });
                inputEl.value = '';
            }
        }

        sendBtn.addEventListener('click', sendMessage);
        inputEl.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });
        clearBtn.addEventListener('click', () => {
            vscode.postMessage({ type: 'clear' });
        });

        window.addEventListener('message', (event) => {
            const data = event.data;
            if (data.type === 'updateMessages') {
                renderMessages(data.messages);
            }
        });
    </script>
</body>
</html>`;
    }
}
