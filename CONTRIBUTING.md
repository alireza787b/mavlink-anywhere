# Contributing

Contributions should preserve the field-safe routing model:

- keep dashboard and CLI workflows simple enough for non-developer operators
- do not expose unsafe defaults on public interfaces
- preserve node-local hardware overlays unless an operator explicitly changes
  them
- update docs and tests with behavior changes
- keep MDS integration optional; this project must remain useful standalone

Before opening a PR, run the relevant tests and include the command output in
the PR description.

