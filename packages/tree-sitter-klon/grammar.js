// oxlint-disable no-unused-vars
/**
 * @file Klon grammar for tree-sitter
 * @author ProCode Software
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const unicodeDigit = /\p{N}/u
const letter = /[\p{L}_]/u

export default grammar({
    name: 'klon',
    extras: $ => [/\s/, $.comment],
    rules: {
        // TODO: add the actual grammar rules
        source_file: $ => 'hello',
        comment: $ =>
            token(choice(seq('//', /.*/), seq('/*', /[^*]*\*+([^/*][^*]*\*+)*/, '/'))),

        int_literal: $ => choice($.hex_literal, $.decimal_literal, $.binary_literal),
        hex_literal: () => /0x[A-Fa-f0-9_]+/,
        binary_literal: () => /0b[0-1_]+/,
        decimal_literal: () => /[0-9_]+/,
        // TODO: Float
    },
})
