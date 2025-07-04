import type { Repository, TextMateLanguage } from 'vsxtools/tmLanguage'
import { include, match, merge } from 'vsxtools/tmLanguage'

RegExp.prototype.toString = function () {
    return this.source
}

const repository = {
    comments: {
        patterns: [
            {
                begin: /\/{2}/,
                end: /$/,
                name: 'comment.line.double-slash.klarmarkup',
            },
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klarmarkup',
            },
        ],
    },
    properties: {
        begin: /^\s*(-*)\s*(?:([-\p{L}\w._/+$@]+)\s*(:)\s*)?/u,
        end: /$/,
        beginCaptures: [
            { name: 'punctuation.definition.block.sequence.item.klarmarkup' },
            { name: 'support.type.property-name.klarmarkup' },
            { name: 'punctuation.separator.key-value.klarmarkup' },
            { include: '#values' },
        ],
        patterns: [include('values')],
    },
    stringLiterals: {
        begin: /("|')/,
        end: '\\1',
        beginCaptures: [{ name: 'punctuation.definition.string.begin.klarmarkup' }],
        endCaptures: [{ name: 'punctuation.definition.string.end.klarmarkup' }],
        contentName: 'string.quoted.klarmarkup',
        patterns: [include('strings')],
    },
    namespaces: match(/@[\p{L}\w\d_.+-]+/u, 'support.class.klarmarkup'),
    strings: {
        patterns: [match(/\\./, 'constant.character.escape.klarmarkup')],
    },
    rawStrings: {
        match: /.+?/,
        name: 'string.unquoted.klarmarkup',
    },
    numbers: {
        patterns: [
            match(/\bv[\d.]+(?:-\w+(?:-\d+)?)?\b/, 'constant.numeric.version.klarmarkup'),
            match(/\b(true|false)\b/, 'constant.language.boolean.$1.klarmarkup'),
            match(
                /(?:\b|[-+])[\d_]+(?:.[\d_]+)?\b/,
                'constant.numeric.decimal.klarmarkup'
            ),
            match(/(?:\B|[-+])\.[\d_]+\b/, 'constant.numeric.decimal.klarmarkup'),
        ],
    },
    operators: {
        patterns: [
            match(/[><]=?/, 'keyword.operator.relational.klarmarkup'),
            match(/\|/, 'keyword.operator.relational.klarmarkup'),
            match(/\.\.[.<]/, 'keyword.operator.range.klarmarkup'),
        ],
    },
    array: match(/,/, 'punctuation.separator.comma.klarmarkup'),
    variableDeclarations: {
        begin: /^\s*(\$)([-\p{L}\w_]+)\s*(=)/u,
        end: /$/,
        beginCaptures: [
            { name: 'punctuation.definition.variable.klarmarkup' },
            { name: 'variable.other.klarmarkup' },
            { name: 'keyword.operator.assignment.klarmarkup' },
        ],
        patterns: [include('values')],
    },
    variables: {
        patterns: [
            {
                match: /(\$)(\.?[-\p{L}\w_]+)/u,
                captures: [
                    { name: 'punctuation.definition.variable.klarmarkup' },
                    { name: 'variable.other.klarmarkup' },
                ],
            },
            {
                match: /(\$\{)\s*(\.?[-\p{L}\w_]+)\s*(\})/u,
                captures: [
                    { name: 'punctuation.definition.variable.klarmarkup' },
                    { name: 'variable.other.klarmarkup' },
                    { name: 'punctuation.definition.variable.klarmarkup' },
                ],
            },
        ],
    },
    values: {
        patterns: [
            'operators',
            'numbers',
            'namespaces',
            'variables',
            'stringLiterals',
            'rawStrings',
            'array',
        ].map(include),
    },
} satisfies Repository

export default {
    name: 'Klar Markup',
    scopeName: 'source.klarmarkup',
    patterns: ['comments', 'variableDeclarations', 'properties', 'values'].map(include),
    repository,
} satisfies TextMateLanguage
