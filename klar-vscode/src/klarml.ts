import type { Repository, TextMateLanguage } from 'vsxtools/tmLanguage'
import { include, match } from 'vsxtools/tmLanguage'

RegExp.prototype.toString = function () {
    return this.source
}

const comma = match(/,/, 'punctuation.separator.comma.klarmarkup')

const repository = {
    commentInside: match(/TODO/, 'keyword.todo.klar'),
    comments: {
        patterns: [
            {
                begin: /\/{2}/,
                end: /$/,
                name: 'comment.line.double-slash.klarmarkup',
                patterns: [include('commentInside')],
            },
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klarmarkup',
                patterns: [include('commentInside')],
            },
        ],
    },
    properties: {
        begin: /(?:(?<=\{)|^)\s*(-*)\s*(?:(\$\s*)?('(?:.*)'|"(?:.*)"|[-\p{L}\w._/+\\]+)\s*(:)\s*)?/u,
        end: /$|(?=})/,
        beginCaptures: [
            { name: 'punctuation.definition.block.sequence.item.klarmarkup' },
            { name: 'punctuation.definition.variable.klar' },
            {
                name: 'support.type.property-name.klarmarkup',
                patterns: [include('keys')],
            },
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
    namespaces: {
        match: /(@)[\p{L}\w\d_.\\+-]+/u,
        name: 'support.class.klarmarkup',
        captures: [undefined, { name: 'punctuation.definition.class.klarmarkup' }],
    },
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
    array: {
        begin: /\[/,
        end: /\]/,
        captures: [{ name: 'punctuation.definition.array.klarmarkup' }],
        patterns: [comma, { include: '#values' }],
    },
    objects: {
        begin: /{/,
        end: /}/,
        captures: [{ name: 'punctuation.definition.object.klarmarkup' }],
        patterns: [
            comma,
            match(/}/, 'punctuation.definition.object.klarmarkup'),
            include('properties'),
        ],
    },
    variables: {
        patterns: [
            {
                match: /(\$)(\.?[-\p{L}\w_\\]+)/u,
                captures: [
                    { name: 'punctuation.definition.variable.klarmarkup' },
                    { name: 'variable.other.klarmarkup' },
                ],
            },
            {
                match: /(\$\{)\s*(\.?[-\p{L}\w_\\]+)\s*(\})/u,
                captures: [
                    { name: 'punctuation.definition.variable.klarmarkup' },
                    { name: 'variable.other.klarmarkup' },
                    { name: 'punctuation.definition.variable.klarmarkup' },
                ],
            },
        ],
    },
    keys: {
        patterns: [
            include('stringLiterals'),
            include('numbers'),
            match('/', 'punctuation.separator.accessor.klarmarkup'),
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
    name: 'Klar Markup',
    scopeName: 'source.klarmarkup',
    patterns: ['comments', 'properties', 'values'].map(include),
    repository,
} satisfies TextMateLanguage
