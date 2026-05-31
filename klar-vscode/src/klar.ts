import type { Pattern, Repository, TextMateLanguage } from 'vsxtools/tmLanguage'
// @ts-ignore
import { include, match, merge } from 'vsxtools/tmLanguage'

// TODO: lambdas in both expressions and types

RegExp.prototype.toString = function () {
    return this.source
}

const Punctuation = {
    period: match(/\./, 'punctuation.separator.period.klar'),
    comma: match(/,/, 'punctuation.separator.comma.klar'),
    colonType: { name: 'keyword.operator.type.annotation.klar' },
    bracket: 'punctuation.definition.bracket',
    at: 'punctuation.definition.attribute.klar',
    generic: 'punctuation.definition.generic.klar',
    equalSign: 'keyword.operator.assignment.klar',
    brace: {
        mapBegin: 'punctuation.definition.map.begin.klar',
        begin: 'punctuation.definition.block.begin.klar',
        end: 'punctuation.definition.block.end.klar',
    },
    parenthesis: {
        begin: { name: 'punctuation.definition.arguments.begin.klar' },
        end: { name: 'punctuation.definition.arguments.end.klar' },
    },
}

const Identifier = /[\p{L}_][\p{L}\w_]*/u,
    IdCapture = /([\p{L}_][\p{L}\w_]*)/u,
    Type = /([-\s\p{L}\w._,?|<>\[\]\-():]+)/u

const IncludeType = { name: 'entity.name.type.klar', patterns: [include('types')] },
    BASE = [{ include: '$base' }],
    EXPR = [{ include: 'expressions' }],
    COMMENTS = [{ include: 'comments' }]

const repository: Repository = {
    comments: {
        patterns: [
            {
                begin: '\\A#!',
                end: /$/,
                name: 'comment.line.shebang.klar',
            },
            {
                begin: /\/{2}/,
                end: /$/,
                name: 'comment.line.double-slash.klar',
                patterns: [include('commentInside')],
            },
            {
                begin: /\/\*/,
                end: /\*\//,
                name: 'comment.block.klar',
                patterns: [include('commentInside'), include('comments')],
            },
        ],
    },
    commentInside: {
        patterns: [
            match(/TODO/, 'keyword.todo.klar'),
            {
                begin: /^\s*```(?:.*)\s*$/,
                end: /^\s*```\s*$|(?=\*\/)/,
                contentName: 'meta.comment.codeblock.klar source.embedded.klar',
                patterns: BASE,
            },
            {
                match: /(\[)([^\]]+)(\])/,
                captures: [
                    { name: 'punctuation.definition.link.markup.klar' },
                    { name: 'string.other.link.title.klar' },
                    { name: 'punctuation.definition.link.markup.klar' },
                ],
            },
            {
                match: /(\*\*|__)([^\s*_].*?)(\1)/,
                captures: [
                    { name: 'punctuation.definition.bold.markup.klar' },
                    { name: 'markup.bold.markup.klar' },
                    { name: 'punctuation.definition.bold.markup.klar' },
                ],
            },
            {
                match: /(\*|_)([^\s*_].*?)(\1)/,
                captures: [
                    { name: 'punctuation.definition.italic.markup.klar' },
                    { name: 'markup.italic.markup.klar' },
                    { name: 'punctuation.definition.italic.markup.klar' },
                ],
            },
            {
                match: /(~~)([^\s~].*?)(\1)/,
                captures: [
                    { name: 'punctuation.definition.strikethrough.markup.klar' },
                    { name: 'markup.strikethrough.markup.klar' },
                    { name: 'punctuation.definition.strikethrough.markup.klar' },
                ],
            },
        ],
    },
    strings: {
        patterns: [
            [/(#+)"/, 'double', [], `"\\1`],
            [/(#+)'/, 'single', [], `'\\1`],
            [/(#+)`/, 'raw', [], '`\\1'],
            [/"/, 'double', ['stringEscape', 'stringInterpolation']],
            [/'/, 'single', ['stringEscape']],
            [/`/, 'raw', ['stringInterpolation']],
        ].map(([b, name, pat, end]) => ({
            begin: b,
            end: end ?? b,
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
        // List of keywords: https://github.com/ProCode-Software/klar/blob/main/internal/lexer/token_types.go#L81
        // Some that are part of declarations aren't present here
        patterns: [
            match(/\b(public)\b/, 'storage.modifier.klar'),
            match(/\b(type)\b/, 'storage.type.type.klar'),
            match(/\b(func)\b/, 'storage.type.function.klar'),
            match(/\b(for|while|next|stop)\b/, 'keyword.control.loop.klar'),
            match(/\b(return|go|await|try)\b/, 'keyword.control.flow.klar'),
            match(/\b(when|import|if)\b/, 'keyword.control.$1.klar'),
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
            match(/(!|\b)in\b/, 'keyword.operator.relational.klar'),
            match(/\|>/, 'keyword.operator.pipe.klar'),
            match(/\|\./, 'keyword.operator.pipe.object.klar'),
            match(/->/, 'keyword.operator.arrow.klar'),
            match(/\.{3}/, 'keyword.operator.spread.klar'),
            match(/\.\.</, 'keyword.operator.range.klar'),
            match(/&&|\|{2}|!/, 'keyword.operator.logical.klar'),
            match(/[><=!]=|[<>]/, 'keyword.operator.comparison.klar'),
            match(/=~|!~/, 'keyword.operator.comparison.regex.klar'),
            match(/\b(and|or)\b/, 'keyword.operator.distributive.klar'),
            match(/[|?]/, 'keyword.operator.type.klar'),
            match(/\+\+/, 'keyword.operator.increment.klar'),
            match(/--/, 'keyword.operator.decrement.klar'),
            match(/[-+:]?=/, Punctuation.equalSign),
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
            Punctuation.parenthesis.begin,
        ],
        endCaptures: [Punctuation.parenthesis.end],
        patterns: [include('labels'), ...BASE],
    },
    builtinFunctions: {
        patterns: [
            match(
                /\b(print|crashout|clone|TODO)\b/,
                'support.function.builtin.klar'
            ),
            match(/\b\p{Lu}[_\p{L}\w]*\b/u, 'entity.name.type.init.klar'),
        ],
    },
    castFunctions: {
        begin: /(?:(\b(?:String|Int|Float|Bool|Error|List|Map|RegEx|Any)\b\??)|\[\g<1>\])(\()/,
        end: /\)/,
        beginCaptures: [
            {
                ...IncludeType,
                name: 'support.type.builtin.klar support.type.primitive.klar',
            },
            IncludeType,
            Punctuation.parenthesis.begin,
        ],
        endCaptures: [Punctuation.parenthesis.end],
        patterns: BASE,
        name: 'meta.type-cast.klar',
    },
    functionDeclarations: {
        begin: merge(
            /(?<=\bfunc\b)\s*/,
            `(?:(?:${IdCapture.source}(\\.))?${IdCapture.source})`,
            /(<[\p{L}\w_,\s]+>)?/u,
            /\s*(\()/
        ),
        end: /(\))(?:\s*(->)\s*TYPE)?/.source.replace('TYPE', Type.source),
        beginCaptures: [
            { name: 'entity.name.type.struct.klar' },
            { name: Punctuation.period.name },
            { name: 'entity.name.function.klar' },
            {
                name: 'meta.function.generic.klar entity.name.type.klar',
                patterns: [match(/[<>]/, Punctuation.generic), Punctuation.comma],
            },
            Punctuation.parenthesis.begin,
        ],
        endCaptures: [
            Punctuation.parenthesis.end,
            { name: 'keyword.operator.return-type.klar' },
            IncludeType,
        ],
        patterns: [
            include('comments'),
            {
                match: /(?<=^|[(,])\s*(\bident\b\s*)?(\bident)\s*(?:(:)\s*type(?:\s*(=)(.*?))?)?\s*(?=\)|,)/.source
                    .replaceAll('ident', Identifier.source)
                    .replaceAll('type', Type.source),
                captures: [
                    { name: 'entity.other.attribute-name.klar' },
                    { name: 'variable.parameter.klar' },
                    Punctuation.colonType,
                    IncludeType,
                    { name: Punctuation.equalSign },
                    { patterns: BASE },
                ],
                patterns: BASE,
            },
            Punctuation.comma,
        ],
    },
    types: {
        patterns: [
            match(/\bfunc\b/, 'storage.type.function.klar'),
            {
                match: merge(Identifier, '(?=\\.)'),
                name: 'entity.name.namespace.klar',
            },
            match(
                /\b(String|Int|Float|Bool|Result|List|Map|Any|Nothing|Error)(?!\.)\b/,
                'support.type.builtin.klar support.type.primitive.klar'
            ),
            {
                match: `(?:\\s*${IdCapture}(,)?\\s*)+\\s*(:)`,
                captures: [
                    { name: 'variable.parameter.klar' },
                    { name: Punctuation.comma.name },
                    Punctuation.colonType,
                ],
            },
            match(/[+|?]/, 'keyword.operator.type.klar'),
            match(/\.{3}/, 'keyword.operator.spread.klar'),
            match(/->/, 'keyword.operator.arrow.klar'),
            match(/[<>]/, Punctuation.generic),
            match(/[\[\]]/, 'punctuation.definition.type.list.klar'),
            match(/[()]/, 'punctuation.definition.type.tuple.klar'),
            Punctuation.comma,
            Punctuation.period,
        ],
    },
    variables: {
        patterns: [
            match(/\b(self)\b/, 'variable.language.self.klar'),
            match(/\b_?[\p{Lu}_][\p{Lu}\d_]*\b/u, 'variable.other.constant.klar'),
            match(Identifier, 'variable.other.readwrite.klar'),
            {
                match: merge(/(\B\.)\s*/, IdCapture, /\b(?![(.])/),
                captures: [
                    { name: 'punctuation.definition.enum.klar' },
                    { name: 'variable.other.enummember.klar' },
                ],
            },
        ],
    },
    labels: {
        patterns: [
            {
                // Shorthand syntax
                match: merge(
                    /(?<=^|[(,])\s*/,
                    /\s*(:)/,
                    `(${Identifier}\\.)*`,
                    IdCapture
                ),
                captures: [
                    { name: 'punctuation.separator.label.parameter.klar' },
                    { patterns: BASE },
                    { name: 'entity.other.attribute-name.klar' },
                ],
            },
            {
                match: merge(/(?<=^|[(,])\s*/, IdCapture, /\s*(:)/),
                captures: [
                    { name: 'entity.other.attribute-name.klar' },
                    { name: 'punctuation.separator.label.parameter.klar' },
                ],
            },
        ],
    },
    punctuation: {
        patterns: [
            match(/[{}]/, 'punctuation.definition.block.klar'),
            match(/[\[\]]/, Punctuation.bracket),
            Punctuation.comma,
            match(/;/, 'invalid.semicolon.klar'),
            match(/:/, 'punctuation.other.colon.klar'),
            match(/@/, Punctuation.at),
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
                            `(?:${IdCapture}|(\\*))`
                        ),
                        captures: [
                            { name: 'entity.name.type.klar' },
                            { name: 'keyword.operator.import-as.klar' },
                            { name: 'keyword.control.type.klar' },
                            { name: 'entity.name.type.klar' },
                            { name: 'keyword.operator.wildcard.klar' },
                        ],
                    },
                    Punctuation.comma,
                    match(/:/, 'keyword.operator.import-as.klar'),
                    match(/\*/, 'keyword.operator.wildcard.klar'),
                ],
            },
            match(/\./, 'punctuation.accessor.namespace.klar'), // keyword.operator.namespace.klar
            match(/=/, 'keyword.operator.import-alias.klar'),
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
            IncludeType,
        ],
    },
    interfaceTag: {
        begin: merge(
            /(?<=\btype\b)\s*(#)\s*/,
            IdCapture,
            String.raw`\s*(?:(:)\s*${IdCapture})?`
        ),
        end: /$/,
        name: 'meta.type.klar',
        beginCaptures: [
            { name: 'punctuation.definition.interface-type.klar' },
            { name: 'entity.name.type.struct.klar' },
            Punctuation.colonType,
            {
                name: 'entity.name.type.struct entity.other.inherited-type.klar',
                patterns: [include('types')],
            },
            { name: Punctuation.brace.begin },
        ],
    },
    typeDeclarations: {
        name: 'meta.type.klar',
        begin: merge(
            /(?<=\btype\b)\s*(#)?/,
            IdCapture,
            /\s*(<[\p{L}\w_,\s]+>)?\s*/u,
            `(?:(:)\\s*${Type})?`,
            `(?:\\s*(->)\\s*${Type})?`,
            /\s*({)/
        ),
        end: /}/,
        beginCaptures: [
            { name: 'punctuation.definition.interface-type.klar' },
            { name: 'entity.name.type.struct.klar' },
            {
                name: 'meta.function.generic.klar entity.name.type.klar',
                patterns: [match(/[<>]/, Punctuation.generic), Punctuation.comma],
            },
            Punctuation.colonType,
            {
                name: 'entity.name.type.struct entity.other.inherited-type.klar',
                patterns: [include('types')],
            },
            { name: 'keyword.operator.arrow.klar' },
            {
                name: 'entity.name.type.klar entity.other.value-type.klar',
                patterns: [include('types')],
            },
            { name: Punctuation.brace.begin },
        ],
        endCaptures: [{ name: Punctuation.brace.end }],
        patterns: [
            {
                match: `(?<=^|{)\\s*(?:(?:${IdCapture}\\s*(,)\\s*)*${IdCapture}\\s*)(:)\\s*${Type}(?:\\s*(=)(.+)(?=$))?`,
                captures: [
                    { name: 'variable.other.klar' },
                    { name: Punctuation.comma.name },
                    { name: 'variable.other.klar' },
                    Punctuation.colonType,
                    IncludeType,
                    { name: Punctuation.equalSign },
                    { patterns: BASE },
                ],
            },
            include('comments'),
            include('interfaces'),
            include('enums'),
            match(/,/, 'invalid.comma.klar'),
        ],
    },
    enums: {
        patterns: [
            {
                begin: merge(/(\.)\s*/, IdCapture, /\s*(\()/),
                end: /(\))/,
                beginCaptures: [
                    { name: 'punctuation.definition.enum.klar' },
                    { name: 'variable.other.enummember.klar' },
                    Punctuation.parenthesis.begin,
                    { patterns: BASE },
                    Punctuation.parenthesis.begin,
                    { name: 'entity.name.type.klar', patterns: [include('types')] },
                    Punctuation.parenthesis.end,
                ],
                patterns: [include('typeLabels'), Punctuation.comma],
            },
            {
                match: merge(/(\.)\s*/, IdCapture),
                captures: [
                    { name: 'punctuation.definition.enum.klar' },
                    { name: 'variable.other.enummember.klar' },
                ],
            },
            Punctuation.comma,
            {
                begin: /=/,
                end: /$|(?=[,}])/,
                beginCaptures: [{ name: 'keyword.operator.assignment.klar' }],
                patterns: BASE,
            }
        ],
    },
    typeLabels: {
        begin: merge(/(?<=\(|,)\s*/, '(?:', IdCapture, '(:))?'),
        end: /(?=,|\))/,
        beginCaptures: [{ name: 'variable.parameter.klar' }, Punctuation.colonType],
        contentName: 'entity.name.type.klar',
        patterns: [include('types')],
    },
    interfaces: {
        begin: `(?<=^|{)\\s*(${Identifier.source})\\s*(\\()`,
        end: String.raw`(\))\s*(?!\|)(?:(->)\s*${Type})?`,
        beginCaptures: [
            { name: 'entity.name.function.member.klar' },
            Punctuation.parenthesis.begin,
        ],
        endCaptures: [
            Punctuation.parenthesis.begin,
            { name: 'keyword.operator.arrow.klar' },
            IncludeType,
        ],
        contentName: 'entity.name.type.klar',
        patterns: [include('labels'), include('types')],
    },
    variableAssignments: {
        begin: merge(
            `(?:(?:${IdCapture}|_|(\\[.*\\]|\\(.*\\)|#\\{.*\\}))(,)?)+`,
            String.raw`(?:\s*(:)\s*${Type})?`,
            /\s*(:=)/
        ),
        beginCaptures: [
            {
                name: 'variable.other.assignment.klar',
                patterns: [include('variables')],
            },
            {
                patterns: [include('destructuring')],
            },
            Punctuation.comma,
            Punctuation.colonType,
            IncludeType,
            { name: 'keyword.operator.assignment.klar' },
        ],
        patterns: BASE,
        end: /$/,
    },
    destructuring: {
        patterns: [
            ...[
                [/\[/, /\]/, Punctuation.brace.begin, Punctuation.brace.end],
                [
                    /\(/,
                    /\)/,
                    Punctuation.parenthesis.begin.name,
                    Punctuation.parenthesis.end.name,
                ],
                [/\#\{/, /\}/, Punctuation.brace.mapBegin, Punctuation.brace.end],
            ].map(
                ([begin, end, beginName, endName]): Pattern => ({
                    begin,
                    end: merge(end, /\s*(?=[-:+]=|:)/),
                    beginCaptures: [{ name: beginName as string }],
                    endCaptures: [{ name: endName as string }],
                    patterns: [
                        {
                            begin: /=/,
                            end: /(?=,|\}|\)|\])/,
                            beginCaptures: [{ name: 'keyword.operator.assignment.klar' }],
                            patterns: [include('expressions')],
                        },
                        match(/:/, Punctuation.colonType.name),
                        include('$self'),
                    ],
                })
            ),
        ],
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
            match(/\?/, 'constant.language.nil.klar'),
            {
                begin: /((?:!|\b)can\b)/,
                end: /(?=[!,]|->|when)/,
                beginCaptures: [{ name: 'keyword.operator.relational.klar' }],
                patterns: [include('functions'), include('types')],
            },
            ...BASE,
        ],
    },
    attributes: {
        begin: merge('(@)', IdCapture, /(\()?/),
        end: /(\))|$/,
        beginCaptures: [
            { name: Punctuation.at },
            { name: 'storage.modifier.attribute.klar' },
            Punctuation.parenthesis.begin,
        ],
        endCaptures: [Punctuation.parenthesis.end],
        name: 'meta.attribute.klar',
        patterns: [include('labels'), ...BASE],
    },
    regex: {
        begin: /(?<!\w\s*)\//,
        end: /(\/)([a-z]*)/,
        beginCaptures: [{ name: 'punctuation.definition.regexp.begin.klar' }],
        endCaptures: [
            { name: 'punctuation.definition.regexp.end.klar' },
            { name: 'keyword.other.regexp.flag.klar' },
        ],
        name: 'string.regexp.klar',
        patterns: [{ include: 'source.js.regexp' }],
    },
    expressions: {
        patterns: [
            'comments',
            'regex',
            'strings',
            'booleans',
            'operators',
            'numbers',
            'castFunctions',
            'whenExpression',
            'functions',
            'variables',
            'punctuation',
        ].map(include),
    },
} satisfies Repository

const klar: TextMateLanguage = {
    name: 'Klar',
    patterns: [
        include('comments'),
        include('regex'),
        include('strings'),
        include('booleans'),
        include('lambdas'),
        include('keywords'),
        include('operators'),
        include('numbers'),

        include('typeDeclarations'),
        include('interfaceTag'),
        include('castFunctions'),
        include('functionDeclarations'),
        include('interfaceTag'),
        include('typeAliasDeclarations'),
        include('importStatements'),
        include('variableAssignments'),
        include('whenExpression'),

        include('attributes'),
        include('functions'),
        include('variables'),
        include('punctuation'),
    ],
    repository,
    scopeName: 'source.klar',
}

export default klar
