import { merge, type Pattern, type TextMateLanguage } from 'vsxtools/tmLanguage'

export default {
    scopeName: 'markdown.klar.codeblock',
    injectionSelector: 'L:text.html.markdown',
    patterns: [
        ['klar', 'klar'],
        ['klon', 'klon'],
    ].map(
        ([name, aliases]) =>
            ({
                begin: merge(
                    /(^|\G)(\s*)(\`{3,}|~{3,})\s*/,
                    `(?i:(${aliases})(s+[^\`~]*)?$)`
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
                        contentName: `meta.embedded.block.${name}`,
                        patterns: [{ include: `source.${name}` }],
                    },
                ],
            }) as Pattern
    ),
} satisfies TextMateLanguage
