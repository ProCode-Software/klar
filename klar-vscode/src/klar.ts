import type { Repository, TextMateLanguage } from 'vsxtools/tmLanguage'
import { include, match, merge } from 'vsxtools/tmLanguage'

const BASE = [{ include: '$base' }]

RegExp.prototype.toString = function () {
    return this.source
}

// keywords:
// import  func  type  return  next  for  when

const Punctuation = {
    period: match(/\./, 'punctuation.separator.period.klar'),
    comma: match(/,/, 'punctuation.separator.comma.klar'),
    colonType: 'keyword.operator.type.annotation.klar',
    bracket: 'punctuation.definition.bracket',
    equalSign: 'keyword.operator.assignment.klar',
    brace: {
        begin: 'punctuation.definition.block.begin.klar',
        end: 'punctuation.definition.block.end.klar',
    },
    parenthesis: {
        begin: 'punctuation.definition.arguments.begin.klar',
        end: 'punctuation.definition.arguments.end.klar',
    },
}

const Identifier = /_?[\p{L}_][\p{L}\w_]*/u
const IdCapture = /(_?[\p{L}_][\p{L}\w_]*)/u
//@ts-ignore
const Type: string =
    /(((?:_?[\p{L}_][\p{L}\w_]*\.)?_?[\p{L}_][\p{L}\w_]*(?:<[\s\p{L}\w._,?<>\[\]\-()]+>)?\??)|\[\s*REC\s*\]\??)/u.source.replaceAll(
        'REC',
        '\\g<2>'
    ) // Supports namespaces

const IncludeType = [
    { name: 'entity.name.type.klar', patterns: [include('types')] },
    undefined,
]

const repository: Repository = {
    comments: {
        patterns: [
            {
                begin: /\/{2}/,
                end: /$/,
                name: 'comment.line.double-slash.klar',
            },
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klar',
            },
        ],
    },
    strings: {
        patterns: [
            [/"/, 'double', ['stringEscape', 'stringInterpolation']],
            [/'/, 'single', ['stringEscape']],
            [/`/, 'raw', []],
        ].map(([b, name, pat]) => ({
            begin: b,
            end: b,
            name: `string.quoted.${name}.klar`,
            beginCaptures: [{ name: 'punctuation.definition.string.begin.klar' }],
            endCaptures: [{ name: 'punctuation.definition.string.end.klar' }],
            patterns: (pat as string[]).map(n => include(n)),
        })),
    },
    stringEscape: {
        patterns: [
            {
                match: /\\(?:x[0-9a-fA-F]{2}|u{0*[0-9a-fA-F]{1,6}}|u[0-9a-fA-F]{4}|.)/,
                name: 'constant.character.escape.klar',
            },
        ],
    },
    stringInterpolation: {
        begin: /{/,
        end: /}/,
        contentName: 'meta.interpolated-string.klar source.klar',
        patterns: BASE,
        captures: [{ name: 'punctuation.definition.string-interpolation.klar' }],
    },
    keywords: {
        patterns: [
            match(/\b(type)\b/, 'storage.type.type.klar'),
            match(/\b(func)\b/, 'storage.type.function.klar'),
            match(/\b(for|next)\b/, 'keyword.control.loop.klar'),
            match(/\b(return)\b/, 'keyword.control.flow.klar'),
            match(/\b(when|import)\b/, 'keyword.control.$1.klar'),
        ],
    },
    booleans: {
        patterns: [
            match(/\b(true|false)\b/, 'constant.language.boolean.$1.klar'),
            match(/\b(nil)\b/, 'constant.language.nil.klar'),
        ],
    },
    numbers: {
        patterns: [
            match(/\b0[xX][0-9A-Fa-f_]+\b/, 'constant.numeric.hex.klar'),
            match(/\b0[oO][0-7_]+\b/, 'constant.numeric.octal.klar'),
            match(/\b0[bB][0-1]+\b/, 'constant.numeric.binary.klar'),
            match(
                /\b[\d_]+(?:e[+-]?[\d_]+)?\.[\d_]+(?:e[+-]?[\d_]+)?\b/, // 3e+4.3e2
                'constant.numeric.decimal.klar'
            ),
            match(/\.[\d_]+(?:e[+-]?[\d_]+)?\b/, 'constant.numeric.decimal.klar'), // .34e2
            match(/\b[\d_]+(?:e[+-]?[\d_]+)?\./, 'constant.numeric.decimal.klar'), // 3e3.
            match(/[\d_]+(?:e[+-]?[\d_]+)/, 'constant.numeric.decimal.klar'), // 3e2
            match(/\b(?:\d+_?)+\b/, 'constant.numeric.decimal.klar'), // 97_468
            /* match(
                /\b[\d_]+(?:e[+-]?[\d_]+)?\.[\d_]+(?:e[+-]?[\d_]+)?\b|\.[\d_]+(?:e[+-]?[\d_]+)?\b|\b[\d_]+(?:e[+-]?[\d_]+)?\.|[\d_]+(?:e[+-]?[\d_]+)|\b(?:\d+_?)+\b/,
                'constant.numeric.decimal.klar'
            ), */
        ],
    },
    operators: {
        patterns: [
            match(/\|>/, 'keyword.operator.pipe.klar'),
            match(/->/, 'keyword.operator.arrow.klar'),
            match(/\.{3}/, 'keyword.operator.spread.klar'),
            match(/\.{2}/, 'keyword.operator.range.klar'),
            match(/&&|\|{2}|!/, 'keyword.operator.logical.klar'),
            match(/[><=!]=|[<>]/, 'keyword.operator.comparison.klar'),
            match(/[&|?]/, 'keyword.operator.type.klar'),
            match(/[-+*/%^:]?=/, Punctuation.equalSign),
            match(/[-+*/%^]/, 'keyword.operator.arithmetic.klar'),
            match(/=/, Punctuation.equalSign),
        ],
    },
    functions: {
        begin: merge(IdCapture, /\s*(\()/),
        end: /\)/,
        name: 'meta.function-call.klar',
        beginCaptures: [
            {
                name: 'entity.name.function.klar',
                patterns: [include('builtinFunctions')],
            },
            { name: Punctuation.parenthesis.begin },
        ],
        endCaptures: [{ name: Punctuation.parenthesis.end }],
        patterns: [include('labels'), ...BASE],
    },
    builtinFunctions: match(/\b(print|panic|TODO)\b/, 'support.function.builtin.klar'),
    castFunctions: {
        begin: /(?:(\b(?:String|Int|Float|Bool|Error|List|Dictionary)\b\??)|\[TYPE\])(\()/.source.replaceAll(
            'TYPE',
            Type
        ),
        end: /\)/,
        beginCaptures: [
            {
                ...IncludeType[0],
                name: 'support.type.builtin.klar support.type.primitive.klar',
            },
            ...IncludeType,
            undefined,
            { name: Punctuation.parenthesis.begin },
        ],
        endCaptures: [{ name: Punctuation.parenthesis.end }],
        patterns: BASE,
        name: 'meta.type-cast.klar',
    },
    functionDeclarations: {
        begin: merge(
            /(?<=\bfunc\b)\s*/,
            `(?:(?:${IdCapture.source}(\\.))?${IdCapture.source})?`,
            /\s*(\()/
        ),
        end: /(\))(?:\s*(->)\s*TYPE)?/.source.replace('TYPE', Type),
        beginCaptures: [
            { name: 'entity.name.type.struct.klar' },
            { name: Punctuation.period.name },
            { name: 'entity.name.function.klar' },
            { name: Punctuation.parenthesis.begin },
        ],
        endCaptures: [
            { name: Punctuation.parenthesis.end },
            { name: 'keyword.operator.return-type.klar' },
            ...IncludeType,
        ],
        patterns: [
            {
                match: `(?<=^|[(,])\\s*(\\b${Identifier.source}\\b)?\\s*(${Identifier.source})\\s*(:)\\s*${Type}`,
                captures: [
                    { name: 'variable.parameter.klar' },
                    { name: 'variable.other.klar' },
                    { name: Punctuation.colonType },
                    ...IncludeType,
                ],
            },
        ],
    },
    types: {
        patterns: [
            {
                match: merge(Identifier, '(?=\\.)'),
                name: 'entity.name.namespace.klar',
            },
            match(
                /\b(String|Int|Float|Bool|Result|List|Dictionary|Any|Never|Nothing|Error)(?!\.)\b/,
                'support.type.builtin.klar support.type.primitive.klar'
            ),
            match(/[&|?]/, 'keyword.operator.type.klar'),
            match(/[<>]/, 'punctuation.definition.generic.klar'),
            match(/[\[\]]/, 'punctuation.definition.type.array.klar'),
            Punctuation.comma,
            Punctuation.period,
        ],
    },
    variables: {
        patterns: [
            match(/\b(self)\b/, 'variable.language.self.klar'),
            match(/_?[\p{Lu}_][\p{Lu}\d_]*/u, 'variable.other.constant.klar'),
            match(Identifier, 'variable.other.readwrite.klar'),
            {
                match: merge(/(\B\.)/, IdCapture),
                captures: [
                    { name: 'punctuation.definition.enum.klar' },
                    { name: 'variable.other.enummember.klar' },
                ],
            },
        ],
    },
    labels: {
        match: merge(/(?<=^|[(,])\s*/, IdCapture, /\s*(:)/),
        captures: [
            { name: 'variable.parameter.klar' },
            { name: 'punctuation.separator.label.parameter.klar' },
        ],
    },
    punctuation: {
        patterns: [
            match(/[{}]/, 'punctuation.definition.block.klar'),
            match(/[\[\]]/, Punctuation.bracket),
            Punctuation.comma,
            match(/;/, 'invalid.semicolon.klar'),
            Punctuation.period,
        ],
    },
    importStatements: {
        begin: /(?<=\bimport\b)/,
        beginCaptures: [{ patterns: BASE }],
        end: /$/,
        name: 'meta.import.klar',
        contentName: 'entity.name.namespace.klar',
        patterns: [
            {
                begin: /\.\s*{/,
                end: /}/,
                captures: [{ name: 'punctuation.definition.unqualified-import.klar' }],
                contentName: 'variable.other.klar',
                patterns: [
                    {
                        match: merge(
                            `(?:(${Identifier.source})\\s*(:)\\s*)?`,
                            /\b(type)\b\s*/,
                            IdCapture
                        ),
                        captures: [
                            { name: 'entity.name.type.klar' },
                            { name: 'keyword.operator.import-as.klar' },
                            { name: 'keyword.control.type.klar' },
                            { name: 'entity.name.type.klar' },
                        ],
                    },
                    Punctuation.comma,
                    match(/:/, 'keyword.operator.import-as.klar'),
                    match(/\*/, 'keyword.operator.wildcard.klar'),
                ],
            },
            match(/\./, 'punctuation.accessor.namespace.klar'), // keyword.operator.namespace.klar
            match(/:/, 'keyword.operator.import-as.klar'),
            match(/\*/, 'keyword.operator.wildcard.klar'),
        ],
    },
    typeAliasDeclarations: {
        begin: merge(/(?<=\btype\b)\s*/, IdCapture, /\s*(=)\s*/, Type),
        end: /$/,
        name: 'meta.typealias-declaration.klar',
        beginCaptures: [
            { name: 'entity.name.type.klar' },
            { name: Punctuation.equalSign },
            ...IncludeType,
        ],
    },
    structs: {
        name: 'meta.type.klar',
        begin: merge(
            /(?<=\btype\b)\s*/,
            IdCapture,
            String.raw`\s*(?:\(${Type}\))?`,
            /\s*({)/
        ),
        end: /}/,
        beginCaptures: [
            { name: 'entity.name.type.struct.klar' },
            { name: 'entity.other.inherited-type.klar', patterns: [include('types')] },
            undefined,
            { name: Punctuation.brace.begin },
        ],
        endCaptures: [{ name: Punctuation.brace.end }],
        patterns: [
            {
                match: `(?<=^|{)\\s*(${Identifier.source})\\s*(:)\\s*${Type}(?:\\s*(=)(.+)(?=$))?`,
                captures: [
                    { name: 'variable.other.klar' },
                    { name: Punctuation.colonType },
                    ...IncludeType,
                    { name: Punctuation.equalSign },
                    { patterns: BASE },
                ],
            },
            include('enums'),
            match(/,/, 'invalid.comma.klar'),
        ],
    },
    enums: {
        patterns: [
            {
                match: merge(IdCapture, /(?:\s*(=)\s*(.+)\s*(?=$|}|\|))?/),
                captures: [
                    { name: 'variable.other.enummember.klar' },
                    { name: Punctuation.equalSign },
                    { patterns: BASE },
                ],
            },
            match(/\|/, 'keyword.operator.type.klar'),
        ],
    },
    variableAssignments: {
        begin: merge(IdCapture, String.raw`(?:\s*(:)\s*${Type})?`, /\s*(:=)/),
        beginCaptures: [
            {
                name: 'variable.other.assignment.klar',
                patterns: [include('variables')],
            },
            { name: Punctuation.colonType },
            ...IncludeType,
            { name: 'keyword.operator.assignment.klar' },
        ],
        patterns: BASE,
        end: /$/,
    },
    whenExpression: {
        begin: /(?<=\bwhen\b)(.+)({)/,
        end: /}/,
        beginCaptures: [
            { name: 'meta.block.when.expression.klar', patterns: BASE },
            { name: Punctuation.brace.begin },
        ],
        endCaptures: [{ name: Punctuation.brace.begin }],
        name: 'meta.block.when.klar',
        patterns: [
            match(/\|(?!\|)/, 'keyword.operator.alternative.klar'),
            match(/\b_\b/, 'keyword.operator.default.klar'),
            ...BASE,
        ],
    },
} satisfies Repository

const klar: TextMateLanguage = {
    name: 'Klar',
    patterns: [
        include('comments'),
        include('strings'),
        include('booleans'),
        include('keywords'),
        include('operators'),
        include('numbers'),

        include('castFunctions'),
        include('functionDeclarations'),
        include('structs'),
        include('typeAliasDeclarations'),
        include('importStatements'),
        include('variableAssignments'),
        include('whenExpression'),

        include('functions'),
        include('variables'),
        include('punctuation'),
    ],
    repository,
    scopeName: 'source.klar',
}

export default klar
