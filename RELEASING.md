# Releasing

Releases are cut by [release-please](https://github.com/googleapis/release-please),
driven by [Conventional Commits](https://www.conventionalcommits.org/), following
the same setup as [tsuru](https://github.com/tsuru/tsuru) and
[terraform-provider-tsuru](https://github.com/tsuru/terraform-provider-tsuru).

## How it works

1. Every push to `main` runs the `release-please` workflow, which keeps a single
   open pull request titled `chore(main): release <version>`. That PR holds the
   next version number and the generated `CHANGELOG.md` entry.
2. Merging that PR makes release-please push the `v<version>` tag and create a
   **draft** release whose body is the changelog entry.
3. The same workflow then dispatches `go.yaml` on the new tag, which runs the
   tests, builds the `tsuru/client:<version>` image and runs goreleaser.
   goreleaser attaches the `tsuru` archives to the draft release, publishes it,
   and updates the Homebrew tap and the AUR package; packagecloud is published
   right after.

So the release only becomes visible once the tag has gone green through the
whole pipeline. No tokens beyond the ones the `deploy` job already uses are
needed: tags pushed with `GITHUB_TOKEN` do not trigger other workflows, which is
why the tag is handed over through `workflow_dispatch` instead of relying on the
tag push.

Nothing has to be tagged or versioned by hand. The version reported by
`tsuru version` is injected by goreleaser from the tag at build time.

## Commit messages

The version bump is derived from the commits landing on `main`:

- `fix: ...` (and `perf:`, `revert:`) bumps the patch number
- `feat: ...` bumps the minor number
- `feat!: ...` or a `BREAKING CHANGE:` footer bumps the major number
- `chore: ...` and `deps: ...` appear in the notes but do not bump the version
  on their own; this is where dependabot's `chore(deps): bump ...` subjects land
- `ci:`, `docs:`, `test:`, `build:`, `refactor:`, `style:` and unprefixed
  subjects neither bump the version nor appear in the notes

Pull requests are squash-merged with their title as the commit subject, so the
**pull request title** is what release-please reads, and the `pr-title` workflow
rejects titles that do not follow the format. A release only happens once at
least one `fix:`/`feat:` commit has landed since the previous tag — a run of
dependency bumps on its own does not open a release pull request, the bumps ride
along with the next fix or feature. A dependency bump that has to ship on its
own (a CVE, say) should be titled `fix(deps): ...` so that it cuts a patch.

## Recovery

If the dispatch of `go.yaml` fails, the tag and the draft release survive.
Rerun it with `gh workflow run go.yaml --ref <tag>`, or publish the draft by
hand.
