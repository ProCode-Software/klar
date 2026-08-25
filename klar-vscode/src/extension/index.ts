import { existsSync, statSync } from 'fs'
import { isAbsolute, resolve } from 'path'
import { ExtensionContext, commands, window, workspace } from 'vscode'

// Code derived from:
// - https://github.com/gleam-lang/vscode-gleam/blob/main/src/extension.ts
// - https://code.visualstudio.com/api/language-extensions/language-server-extension-guide
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
} from 'vscode-languageclient/node'

let client: LanguageClient | undefined

export async function activate({ subscriptions }: ExtensionContext) {
    const restartCommand = commands.registerCommand('klar.restartServer', async () => {
        if (!client) {
            window.showErrorMessage('KlarLS not found')
            return
        }
        try {
            // oxlint-disable-next-line no-unused-expressions
            client.isRunning() ? await client.restart() : await client.start()
        } catch (err) {
            client.error('Failed to restart Klar language server', err, 'force')
        }
    })
    subscriptions.push(restartCommand)

    const command = await getServerPath()
    if (!command) {
        window.showErrorMessage(
            "Couldn't find a Klar executable. Make sure 'klar' is available in the PATH used by VSCode, or set 'klar.path' to a valid executable."
        )
        return
    }

    const serverOptions: ServerOptions = {
        command,
        args: ['lsp'],
        options: { env: Object.assign(process.env, { NO_COLOR: 1 }) },
    }
    const clientOptions: LanguageClientOptions = {
        documentSelector: ['klar', 'klon', 'glas.lock'].map(language => ({
            language,
            // scheme: 'file', // Disabled so untitled files are supported
        })),
        outputChannel: window.createOutputChannel('Klar', { log: true }),
    }

    client = new LanguageClient(
        'klarLS',
        'Klar Language Server',
        serverOptions,
        clientOptions
    )
    await client.start()
}

export async function deactivate() {
    await client?.dispose()
    client = undefined
}

async function getServerPath(): Promise<string | undefined> {
    let pathSetting: string | undefined = workspace.getConfiguration('klar').get('path')
    if (typeof pathSetting != 'string' || pathSetting.trim() == '')
        pathSetting = undefined
    const isAbsPath = pathSetting && isAbsolute(pathSetting)
    if (isAbsPath) return pathSetting

    // If they provided a relative path, we have to search in each workspace folder
    const workspaceFolders = workspace.workspaceFolders
    if (!pathSetting || !workspaceFolders) return 'klar'

    for (const {
        uri: { fsPath },
    } of workspaceFolders) {
        const cmdPath = resolve(fsPath, pathSetting)
        if (existsSync(cmdPath) && statSync(cmdPath).isFile()) return cmdPath
    }
    return undefined
}
