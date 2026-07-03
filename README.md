<div align="center">

# xocode

**Plan with Opus 4.8. Build with Composer 2.5. From one terminal.**

</div>

xocode is a terminal UI that runs a **plan → review → build** workflow by
orchestrating the CLIs you already trust:

1. **Plan** — chat with Claude Code (`claude`) in read-only plan mode (**Opus 4.8
   at high effort**). It's a real conversation: say _hi_ and it replies; describe
   a change and it researches your code, then presents an implementation plan.
   xocode only advances to review once Claude has produced an actual plan — a
   greeting or a clarifying question is never mistaken for one.
2. **Review** — read the rendered plan, `edit` it in `$EDITOR`, `refine` it by
   asking Claude for changes, or `discard` it. Then approve.
3. **Build** — Cursor (`cursor-agent`) runs **Composer 2.5** to implement the
   plan in an **isolated git worktree**, so your branch stays clean until you
   merge. A live file-change list shows what's being touched.

### In the TUI

- **Enter** sends your message · **Alt+Enter** for a newline
- A progress **stepper** (Task › Plan › Review › Build › Done) and a live status
  bar with elapsed time, token counts, and cost
- **ctrl+r** browse & reopen saved plans · **ctrl+s** settings (model / effort)
- **?** keyboard-shortcut help · mouse-wheel scrolling

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
you ──chat──▶ claude (Opus 4.8, plan mode) ──plan?──▶ .xocode/plans/<ts>-<slug>.md
                    ▲          │                             │
                    └─ no plan ┘                  review / edit / refine / approve
                   (keep chatting)                           │
                                                             ▼
                                    cursor-agent (Composer 2.5, git worktree)
                                                             │
                                                 review · merge · discard
```

Under the hood, xocode appends a small protocol to Claude's system prompt asking
it to wrap a finished plan in sentinel markers. That's how it reliably tells a
real plan apart from ordinary conversation — no guessing on the response text.

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
