import type { ExtensionConfig } from 'vsxtools'

export default {
    configurations: {
        default: {
            type: 'language',
            inputs: ['src/klar.ts', 'src/klarml.ts', 'src/codeblock.ts'],
            outputFile: 'syntaxes/[name].tmLanguage.json'
        }
    },
    watch: true,
    jsonIndent: 2,
} satisfies ExtensionConfig
