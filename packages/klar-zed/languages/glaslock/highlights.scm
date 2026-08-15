(package_directive ("package" @keyword))

["package" "from" "lockfile" "klar"] @keyword

[
"dependency" "branch" "for" "integrity" "path" "registry" "subpath" "tag" "url"
] @property

("for" ("workspace") @property)

(string) @string
(commit_number) @string
(version) @number
(lockfile_directive _) @number
(comment) @comment

(package_url_directive (string) @string @link_uri)
(package_path_directive (string) @string @link_uri)
(package_registry_directive (string) @string @link_uri)
(package_subpath_directive (string) @string @link_text)
(source) @constant.builtin @variant
