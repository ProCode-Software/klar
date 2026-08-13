/**
 * @file The progressive programming language
 * @author ProCode Software
 * @license Apache-2.0
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

export default grammar({
    name: 'klar',

    rules: {
        // TODO: add the actual grammar rules
      source_file: $ => 'hello',
    },
})
