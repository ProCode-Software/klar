// oxlint-disable no-useless-escape
import { include, match } from 'vsxtools/tmLanguage'
import type { Repository } from 'vsxtools/tmLanguage'

RegExp.prototype.toString = function () {
    return this.source
}

const comma = match(/,/, 'punctuation.separator.comma.klon')
/* const unquotedString = {
    begin: /.(?!\/(\/|\*))/,
    end: /$|(?=\/(?:\/|\*))/,
    name: 'string.unquoted.klon',
} */

const repository = {
    comments: {
        patterns: [
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klon',
                patterns: [include('comments')],
            },
            { begin: /\/\//, end: '$', name: 'comment.line.klon' },
            { begin: /\A#!/, end: '$', name: 'comment.line.shebang.klon' },
        ],
    },
    numbers: {
        patterns: [
            match(
                /\G\s*\bv[\d.]+(?:[- ]\w+(?:[- ]\d+)?)?\b/,
                'constant.numeric.version.klon'
            ),
            match(/\G\s*\b(true|false)\b/, 'constant.language.boolean.$1.klon'),
            match(/\G\s*\b(none)\b/, 'constant.language.none.klon'),
            match(
                /\G\s*(?:\b|-)[\d_]*\d(?:\.[\d_]*\d)?\b/,
                'constant.numeric.decimal.klon'
            ),
            match(
                /\G\s*(?:\b|-)0x[a-fA-F0-9]*[a-fA-F0-9]\b/,
                'constant.numeric.hexadecimal.klon'
            ),
            match(/\G\s*(?:\b|-)0b[01]*[01]\b/, 'constant.numeric.hexadecimal.klon'),
        ],
    },
    strings: {
        patterns: [
            [/\G\s*"/, 'double', ['stringEscapes', 'stringInterpolations'], /"/],
            [/\G\s*'/, 'single', ['stringEscapes'], /'/],
            [/\G\s*>(")/, 'double', ['stringEscapes', 'stringInterpolations'], /"/],
            [/\G\s*>(')/, 'single', ['stringEscapes'], /'/],
        ].map(([b, name, pat, end]) => ({
            begin: b,
            end: end ?? b,
            name: `string.quoted.${name}.klar`,
            beginCaptures: [{ name: 'punctuation.definition.string.begin.klar' }],
            endCaptures: [{ name: 'punctuation.definition.string.end.klar' }],
            patterns: (pat as string[]).map(include),
        })),
    },
    stringEscapes: {
        patterns: [
            match(/\\./, 'constant.character.escape.klon'), // Valid: [befnrtv"'$]
            match(/\\u\{[a-f0-9A-F]{2,6}\}/, 'constant.character.escape.klon'),
        ],
    },
    objectEntries: {
        patterns: [
            // Object spreads
            {
                match: /^\s*((?:-\s*)*)\s*(<-)\s*(\$(?:\{)?([\w\p{L}_]+)(?:\})?)/v,
                captures: [
                    { name: 'punctuation.definition.object.klon' },
                    { name: 'keyword.operator.rest.klon' },
                    { patterns: [include('variableRefs')] },
                ],
            },
            // Key-value pairs
            {
                begin: /^\s*((?:-\s*)*)\s*(.+)\s*(:)/,
                end: /$/,
                beginCaptures: [
                    { name: 'punctuation.definition.object.klon' },
                    {
                        name: 'variable.other.property meta.object-key.klon',
                        patterns: [
                            include('strings'),
                            include('numbers'),
                            match(/\./, 'punctuation.separator.keypath.klon'),
                        ],
                    },
                    { name: 'punctuation.separator.colon.klar' },
                ],
                patterns: [
                    include('values'),
                    include('punctuation'),
                    include('unquotedString'),
                ],
            },
            // Block lists
            {
                begin: /^\s*((?:-\s*)+)\s/,
                end: /$/,
                beginCaptures: [
                    undefined,
                    { name: 'punctuation.definition.object.klon' },
                ],
                patterns: [
                    include('values'),
                    include('punctuation'),
                    include('unquotedString'),
                ],
            },
        ],
    },
    variableDefinitions: {
        begin: /^\s*(\$)([\w_\p{L}]+)\s*(:)/v,
        end: /$/,
        beginCaptures: [
            { name: 'punctuation.definition.variable.klon' },
            { name: 'variable.other.klon' },
            { name: 'punctuation.separator.colon.klar' },
        ],
        patterns: [include('values'), include('punctuation'), include('unquotedString')],
    },
    inlineObjects: {
        begin: /{/,
        end: /}/,
        captures: [{ name: 'punctuation.definition.inline-object.klon' }],
        patterns: [
            comma,
            {
                // This begin pattern was very hard to write
                begin: /([^-\s]+?.*?)\s*(:)/,
                end: /$|(?=,|})/,
                beginCaptures: [
                    {
                        name: 'variable.other.property meta.object-key.klon',
                        patterns: [
                            match(/\./, 'punctuation.separator.keypath.klon'),
                            include('values'),
                        ],
                    },
                    { name: 'punctuation.separator.colon.klar' },
                ],
                patterns: [include('values')],
            },
            include('objectEntries'),
        ],
    },
    stringInterpolations: include('variableRefs'),
    variableRefs: {
        patterns: [
            {
                match: /(\$)([\w\p{L}_]+)/v,
                captures: [
                    { name: 'punctuation.definition.variable.klon' },
                    { name: 'variable.other.klon' },
                ],
            },
            {
                match: /(\$\{)([\w\p{L}_]+)(\})/v,
                captures: [
                    { name: 'punctuation.definition.variable.klon' },
                    { name: 'variable.other.klon' },
                    { name: 'punctuation.definition.variable.klon' },
                ],
            },
        ],
    },
    inlineLists: {
        begin: /\[/,
        end: /\]/,
        captures: [{ name: 'meta.brace.square.klon' }],
        patterns: [match(/,/, comma.name), include('values'), include('unquotedString')],
    },
    values: {
        patterns: [
            'inlineLists',
            'strings',
            'numbers',
            'variableRefs',
            'inlineObjects',
        ].map(include),
    },
    punctuation: {
        patterns: [
            comma,
            match(/<-/, 'keyword.operator.rest.klon'),
            match(/\[|\]/, 'meta.brace.square.klon'),
        ],
    },
    unquotedString: {
        ...match(/.+?/, 'string.unquoted.klon'),
        patterns: [include('stringEscapes'), include('stringInterpolations')],
    },
} satisfies Repository

export default {
    name: 'Klon',
    scopeName: 'source.klon',
    patterns: [
        'comments',
        'variableDefinitions',
        'objectEntries',
        'punctuation',
        'values',
        'unquotedString',
    ].map(include),
    repository,
}
