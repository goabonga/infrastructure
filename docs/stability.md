# Stability and deprecation

infrastructure follows [Semantic Versioning](https://semver.org/) per component.
Each component carries its own version and changelog, bumped from Conventional
Commits by [multicz](https://github.com/goabonga/multicz).

## Version bumps

| Commit type | Bump |
| --- | --- |
| `feat` | minor |
| `fix`, `perf`, `revert` | patch |
| `deprecate`, `remove` | minor |
| `feat!` / `BREAKING CHANGE:` | major |

## Deprecation cadence

Public surfaces follow an `n + 2` cadence:

1. **Announce** in release `n + 1` with a `deprecate:` commit. The symbol keeps
   working and emits a deprecation notice.
2. **Remove** in release `n + 2` with a `remove:` commit.

Because the deprecation cycle is the breaking-change announcement, the removal
itself ships in a minor release, not a major one. Reserve `feat!:` /
`BREAKING CHANGE:` for changes that must bypass the deprecation cycle (for
example a security fix that cannot wait).
