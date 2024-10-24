## Pull Request Guidelines

- Commit info should be formatted as `type(scope): info about commit`. (e.g. `fix(xdp): fix the memory leak issue in the XDP program.`)

  1. type: type must be one of [build, chore, docs, feat, fix, perf, refactor, revert, release, test, improvement].

  2. scope: scope must be one of [ebpf, usercomm, dataproc, frontend, ui, docs, build, deploy, other].

  3. header: header must not be longer than 72 characters.

- Make sure that running `go build` outputs the correct files.

- Rebase before creating a PR to keep commit history clear.

- Make sure PRs are created to `dev` branch instead of `main` branch.

- If your PR fixes a bug, please provide a description about the related bug.
