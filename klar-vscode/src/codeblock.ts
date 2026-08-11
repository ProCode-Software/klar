// oxlint-disable no-useless-escape no-useless-backreference
import { type Pattern, type TextMateLanguage } from 'vsxtools/tmLanguage'

// Taken from https://github.com/DanielGavin/ols/blob/master/editors/vscode/syntaxes/codeblock.json
export default {
    scopeName: 'markdown.klar.codeblock',
    injectionSelector: 'L:text.html.markdown',
    patterns: ['klar', 'klon'].map(
        (lang): Pattern => ({
            begin: /(^|\G)(\s*)(`{3,}|~{3,})\s*(?i:(LANG)((\s+|:|,|\{|\?)[^`]*)?$)/.source.replaceAll(
                'LANG',
                lang
            ),
            end: /(^|\G)(\2|\s{0,3})(\3)\s*$/,
            beginCaptures: {
                3: { name: 'punctuation.definition.markdown' },
                4: { name: 'fenced_code.block.language.markdown' },
                5: { name: 'fenced_code.block.language.attributes.markdown' },
            },
            endCaptures: { 3: { name: 'punctuation.definition.markdown' } },
            patterns: [
                {
                    begin: /(^|\G)(\s*)(.*)/,
                    while: /(^|\G)(?!\s*([`~]{3,})\s*$)/,
                    contentName: `meta.embedded.block.${lang}`,
                    patterns: [{ include: `source.${lang}` }],
                },
            ],
        })
    ),
} satisfies TextMateLanguage
