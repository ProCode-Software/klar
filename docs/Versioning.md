# Klar-Style Versioning

## Parts of a Version

Each part of a version consists of numbers. Multiple parts of a version are separated by dots `.`. At least 1 and no more than 4 parts can be in a version. Each part indicates:

1. **Major version** - Indicates versions that introduce features that break compatibility with older versions.
2. **Minor version** - Indicates a typical update to a version, such as new features being added. These should not include breaking changes--only new major versions should.
3. **Patch** - Indicates fixes to a current version, such as bug fixes and security patches. These usually should not include new features; just fixing current ones.
4. **Revision** - Indicates a release that does not change functionality, such as updates to metadata, documentation, or formatting.

Additionally, a tag can be provided to a version, denoted by a space. Tags indicate the precedence of a version with the same number as another. If no tag is present, the version is considered stable, unless the major number is `0`. See the [Tags](#tags) section for the list of supported tags. An optional number is allowed after a tag, denoted by a space. Out of two versions with matching version numbers and tag names, the version with the higher number following the tag is considered greater.

```
v1.2.3 rc
v1.2.3 beta 2
```

The last part of a version may be optional build metadata, denoted using `+`. Examples of build metadata include commit hashes or target names.

```
v1.2.3+0cf58de
v1.2.3+ubuntu1
```

Letter of any case, numbers, underscores `_`, and hyphens `-` are allowed in build metadata. Letters and numbers may be in any language, as described by the Unicode version of the implementation. Build metadata has no influence on the ordering of versions.

## Tags

Tags are listed from most stable to least. Of 2 versions, the most stable version is considered higher/greater.

1. No tag (aka `stable`) - All features in these versions are considered stable and safe for production, with fixes for bugs and security as they are reported.
2. `rc` - Usually for previews released before a version goes stable. Little new features are added between RC releases, with most versions dedicated to bug fixes and stability improvements.
3. `beta` - Usually for most preview versions, with features nearing completion. Bug fixes and feedback-based changes are the focus on versions with this tag.
4. `alpha` - Usually for the first previews of a version, with new features being added regularly.
5. `dev` - Usually used for nightly builds and versions built from the tip source code.

## Ordering Versions
Versions must be compared by the parts listed below, in the order as listed. The version with the greater number in each part is considered greater. If a part is equal between two versions, the next part should be compared. If a part is present in only one of the two versions, the version that includes the part is considered greater.

1. Major version number
2. Minor version number
3. Patch version number
4. Revision version
5. Version tag, ordered as defined in [Tags](#tags)
6. Number of the tag

## Version Specifiers

Where accepted, version specifiers allow defining compatible ranges of versions. 

Examples of specifiers include:

- `v2.0+` - Allow version `v2.0` or higher
- `v2.x` - Allow any minor `v2` release, but not `v3`
- `v2.1.x` - Allow any patch `v2.1` release, but not `v2.2`
- `v2.1...2.5` - Allow versions from `v2.1` up to (and including) `v2.5`
- `v2.1..<2.5` - Allow versions from `v2.1` and below `v2.5`
- `*` - Allow any version
- `latest [tag]` - Require the latest version available, including versions with tag `[tag]`. If `latest beta` is used, this matches the latest tagged with `beta`, or the latest stable version, whichever is newer. If `[tag]` is not provided, this matches the latest stable version, disallowing any tagged version.