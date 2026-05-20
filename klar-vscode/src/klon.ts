import type { Repository, TextMateLanguage } from 'vsxtools/tmLanguage'
import { include, match } from 'vsxtools/tmLanguage'

RegExp.prototype.toString = function () {
    return this.source
}

const comma = match(/,/, 'punctuation.separator.comma.klon')

const repository = {
    commentInside: match(/TODO/, 'keyword.todo.klon'),
    comments: {
        patterns: [
            {
                begin: /\/{2}/,
                end: /$/,
                name: 'comment.line.double-slash.klon',
                patterns: [include('commentInside')],
            },
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klon',
                patterns: [include('commentInside')],
            },
        ],
    },
    properties: {
        begin: /(?:(?<=\{)|^)\s*((?:-\s*)*)\s*(?:(\$\s*)?('(?:.*)'|"(?:.*)"|[-\p{L}\w._/+\\]+)\s*(:)\s*)?/u,
        end: /$|(?=})/,
        beginCaptures: [
            { name: 'punctuation.definition.block.sequence.item.klon' },
            { name: 'punctuation.definition.variable.klar' },
            { name: 'support.type.property-name.klon', patterns: [include('keys')] },
            { name: 'punctuation.separator.key-value.klon' },
            { include: '#values' },
        ],
        patterns: [include('values')],
    },
    stringLiterals: {
        begin: /("|')/,
        end: '\\1',
        beginCaptures: [{ name: 'punctuation.definition.string.begin.klon' }],
        endCaptures: [{ name: 'punctuation.definition.string.end.klon' }],
        contentName: 'string.quoted.klon',
        patterns: [include('strings')],
    },
    namespaces: {
        match: /(@)[\p{L}\w\d_.\\+-]+/u,
        name: 'support.class.klon',
        captures: [undefined, { name: 'punctuation.definition.class.klon' }],
    },
    strings: { patterns: [match(/\\./, 'constant.character.escape.klon')] },
    rawStrings: { match: /.+?/, name: 'string.unquoted.klon' },
    numbers: {
        patterns: [
            match(/\bv[\d.]+(?:-\w+(?:-\d+)?)?\b/, 'constant.numeric.version.klon'),
            match(/\b(true|false)\b/, 'constant.language.boolean.$1.klon'),
            match(/(?:\b|[-+])[\d_]+(?:.[\d_]+)?\b/, 'constant.numeric.decimal.klon'),
            match(/(?:\B|[-+])\.[\d_]+\b/, 'constant.numeric.decimal.klon'),
        ],
    },
    array: {
        begin: /\[/,
        end: /\]/,
        captures: [{ name: 'punctuation.definition.array.klon' }],
        patterns: [comma, { include: '#values' }],
    },
    objects: {
        begin: /{/,
        end: /}/,
        captures: [{ name: 'punctuation.definition.object.klon' }],
        patterns: [
            comma,
            match(/}/, 'punctuation.definition.object.klon'),
            include('properties'),
        ],
    },
    variables: {
        patterns: [
            {
                match: /(\$)(\.?[-\p{L}\w_\\]+)/u,
                captures: [
                    { name: 'punctuation.definition.variable.klon' },
                    { name: 'variable.other.klon' },
                ],
            },
            {
                match: /(\$\{)\s*(\.?[-\p{L}\w_\\]+)\s*(\})/u,
                captures: [
                    { name: 'punctuation.definition.variable.klon' },
                    { name: 'variable.other.klon' },
                    { name: 'punctuation.definition.variable.klon' },
                ],
            },
        ],
    },
    keys: {
        patterns: [
            include('stringLiterals'),
            include('numbers'),
            match('/', 'punctuation.separator.accessor.klon'),
        ],
    },
    values: {
        patterns: [
            'comments',
            'objects',
            'array',
            'stringLiterals',
            'namespaces',
            'variables',
            'numbers',
            'rawStrings',
        ].map(include),
    },
} satisfies Repository

export default {
    name: 'Klon',
    scopeName: 'source.klon',
    patterns: ['comments', 'properties', 'values'].map(include),
    repository,
} satisfies TextMateLanguage
