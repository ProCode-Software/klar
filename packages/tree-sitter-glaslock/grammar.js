// oxlint-disable no-unused-vars
/**
 * @file Glaslock grammar for tree-sitter
 * @author ProCode Software
 * @license MIT
 * @see Syntax definition: https://github.com/ProCode-Software/klar/tree/main/internal/config/glaslock/parse.go
 */

// TODO: Add explicit newline terminators between directives like tree-sitter-go?

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check
export default grammar({
    name: 'glaslock',
    extras: $ => [/\s/, $.comment],
    word: $ => $.string,
    supertypes: $ => [$.package_subdirective],
    rules: {
        source_file: $ =>
            seq($.lockfile_directive, $.klar_directive, repeat($.full_package_directive)),
        lockfile_directive: () => seq('lockfile', /v?\d+/),
        klar_directive: $ => seq('klar', $.version),

        full_package_directive: $ =>
            seq($.package_directive, repeat1($.package_subdirective)),
        package_directive: $ => seq('package', $.package_header),
        package_header: $ =>
            seq(
                field('name', $.string),
                field('version', $.version),
                'from',
                field('source', $.source)
            ),
        source: $ =>
            choice(
                'npm',
                'local',
                'workspace',
                seq('git', field('commit', $.commit_number))
            ),

        package_subdirective: $ =>
            choice(
                $.package_branch_directive,
                $.package_dependency_directive,
                $.package_for_dev_directive,
                $.package_for_workspace_directive,
                $.package_integrity_directive,
                $.package_path_directive,
                $.package_registry_directive,
                $.package_subpath_directive,
                $.package_tag_directive,
                $.package_url_directive
            ),
        package_branch_directive: $ => seq('branch', $.string),
        package_dependency_directive: $ => seq('dependency', $.package_header),
        package_for_dev_directive: () => seq('for', 'dev'),
        package_for_workspace_directive: $ => seq('for', 'workspace', $.string),
        package_integrity_directive: $ => seq('integrity', $.string),
        package_path_directive: $ => seq('path', $.string),
        package_registry_directive: $ => seq('registry', $.string),
        package_subpath_directive: $ => seq('subpath', $.string),
        package_tag_directive: $ => seq('tag', $.string),
        package_url_directive: $ => seq('url', $.string),

        comment: () => seq('#', /.*/),
        version: () => /v(\d+\.)*\d+/,
        string: () => /[^\s\n#]+/,
        commit_number: () => /[a-f\d]+/,
    },
})
