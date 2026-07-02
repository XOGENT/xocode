<div align="center">

# xocode

**Plan with Opus 4.8. Build with Composer 2.5. From one terminal.**

</div>

xocode is a terminal UI that runs a **plan → review → build** workflow by
orchestrating the CLIs you already trust:

1. **Plan** — Claude Code (`claude`) runs in read-only plan mode with **Opus 4.8
   at high effort** and streams an implementation plan, saved to a plan document.
2. **Review** — read the plan, edit it in `$EDITOR`, and approve.
3. **Build** — Cursor (`cursor-agent`) runs **Composer 2.5** to implement the
   plan in an **isolated git worktree**, so your branch stays clean until you
   merge.

## Install

```sh
curl https://code.xogent.com/install -fsS | bash
```

Then run:

```sh
xocode
```

On first run, xocode checks that the Claude Code and Cursor CLIs are installed
and logged in — and installs / walks you through login if they aren't. Both use
**your own** Claude and Cursor accounts.

### Pin a version

```sh
XOCODE_VERSION=v1.0.0 curl https://code.xogent.com/install -fsS | bash
```

## Commands

| Command          | What it does                                             |
| ---------------- | -------------------------------------------------------- |
| `xocode`         | Launch the interactive TUI (the main way to use it).     |
| `xocode doctor`  | Check prerequisites. `--json` for CI (non-zero on fail). |
| `xocode upgrade` | Update to the latest release. `--check` to only report.  |
| `xocode version` | Print the version.                                       |

## How it works

```
you ──type a task──▶ claude (Opus 4.8, plan mode) ──▶ .xocode/plans/<ts>-<slug>.md
                                                          │
                                              review / edit / approve
                                                          │
                                                          ▼
                                   cursor-agent (Composer 2.5, git worktree)
                                                          │
                                              review · merge · discard
```

- Plans are written to `.xocode/plans/` in your project (override with
  `$XOCODE_PLAN_DIR`).
- The build runs in `~/.cursor/worktrees/<repo>/<name>`; xocode prints the exact
  commands to review, merge, or discard it.

## Requirements

- macOS or Linux (arm64 or amd64)
- A [Claude](https://claude.ai) account and a [Cursor](https://cursor.com)
  account (xocode helps you install the CLIs and log in)

## Building from source

```sh
go build -o xocode ./cmd/xocode
```

## License

[MIT](LICENSE)
