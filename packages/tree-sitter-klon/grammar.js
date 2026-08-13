/**
 * @file Klon grammar for tree-sitter
 * @author ProCode Software
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

export default grammar({
    name: 'klon',
    extras: $ => [/\s/, $.comment],
    rules: {
        // TODO: add the actual grammar rules
        source_file: $ => 'hello',
        comment: $ =>
            token(choice(seq('//', /.*/), seq('/*', /[^*]*\*+([^/*][^*]*\*+)*/, '/'))),
    },
})
