---
name: ghosthost
description: Use when you have produced or identified a local file (video, image, plot, downloaded clip) that the user would benefit from viewing remotely on a phone or another device. Generates a temporary HTTP URL via ghosthost.
---

# ghosthost skill

Use this skill when the user has asked to see a local file on another
device, or when you have produced a file the user cannot easily view
in-terminal (video, image, rendered plot, downloaded media).

## When to use

Invoke `ghosthost share` ONLY when the user has issued an explicit request to share, host, or view a file on another device. Triggers:

- **Explicit verb + file or pronoun referring to a file.** "share this", "host it", "serve this file", "put this somewhere I can see it", "let me see it on my phone", "show me that video", "can you get that onto my phone".
- **Clear viewing intent.** "I want to open this on my phone", "need this on my laptop in the other room", "open it on my tablet".
- **You have just produced a file** (rendered video, generated plot, downloaded clip) that the user explicitly asked for and that they can't view in-terminal — treat this as the user having asked to see it.

## When NOT to use

Do **not** invoke `ghosthost share` when:

- The user mentions a file path as **context**, **input**, or **raw material** for some other task. Examples that look like share requests but are NOT:
  - "Here's the logo: `C:\assets\logo.png`. Embed it in the README."
  - "The demo video is at `C:\Videos\demo.mp4` — add it to the docs."
  - "Read `/home/me/notes.md` and summarize it."
  - "The config file is at `%APPDATA%\ghosthost\config.toml`."
  In all of these, the file path is *scaffolding for a different task*. Do NOT share it.
- The user is pointing at a file so you can **edit it**, **read it**, **analyze it**, **commit it**, or **reference it in code**.
- You have just written a file *internally* (scratch output, a generated test artifact, a build product). Unless the user explicitly asked to view it, don't share.

**Tiebreaker rule: if there is no explicit share/host/view verb in the same message, do not share. Ask instead.** A file path alone is not a share request.

## How to use

1. Run:

       ghosthost --json share <absolute-path>... [--ttl 24h] [--as <name>] [--anon]

2. Parse the JSON response. `--json` always emits a JSON array — one element per file. Required fields per element:

       [{ "schema_version": "1", "id": "...", "url": "...", "expires_at": "..." }]

   For a single file use `result[0].url`. For multiple files iterate the array.

3. Present the URL to the user as a clickable link, with the id and
   expiry timestamp. See "Presenting the URL" below — this matters.

Pass the path the user gave you straight to `ghosthost share`. **Do not
pre-check that the file exists** with `ls`, `Test-Path`, `stat`, `Read`,
`Glob`, or similar. `ghosthost share` validates the path itself (absolute
path, regular file, not a reparse point, not UNC, etc.) and returns exit
code 6 with a clear message if anything is wrong. Pre-checks waste tokens
and can trigger unnecessary OS permission prompts; the CLI is the
authoritative validator.

## Reading share output

The `ghosthost share` command in human mode prints one URL per file, in argv order. If you need the id (for later `revoke`) or the expiry:

- Pass `--json` to get structured output. **`--json` always emits a JSON array** — even for a single file. Parse `result[0].url` for the URL of a single share; iterate the array for multi-file invocations. Example: `jq '.[0].url'`.
- Or call `ghosthost info <arg>` later, where `<arg>` can be the full URL, the URL path (`/t/<token>/<name>`), the bare token, or the bare id. `info` returns the same metadata as `--verbose`-mode share.

`info` returns an error on expired, revoked, or unknown shares — it only resolves currently-live shares.

## Multiple local files

When the user wants to share several files at once, pass them all in a single `ghosthost share` invocation:

    ghosthost share file1.png file2.png file3.png

This avoids multiple permission prompts. Each file gets its own URL on its own line of stdout. `--json` returns a JSON array with one element per file.

- `--as` requires exactly one file; omit it when sharing multiple files.
- Batches over 64 files require `--yes`.
- Validation is atomic: if any path is bad, no shares are created and all errors are reported.

## Anonymize filenames with `--anon`

Pass `--anon` when the source filename might be sensitive. The extension is preserved so the recipient's browser handles the download correctly; only the filename is replaced with a random 6-char slug:

    ghosthost --json share --anon secret-report.pdf

`--anon` works for single-file and multi-file invocations alike.

## Presenting the URL

The whole point of this tool is handing the user a link they can tap. If
the URL isn't clickable in the chat UI, the tool has failed its job.

**Do** emit the URL as a bare URL on its own line, outside any fenced
code block. Most chat UIs (including Claude Code) auto-link bare URLs to
`http://` / `https://` in prose but not inside ``` fences.

**Example — good (URL is clickable):**

> Shared `clip.mp4` (id `8f2b1c04e7a6`, expires 2026-04-20T14:32:08Z).
>
> http://homepc.tail-4a9c2e.ts.net:8750/t/k3n.../clip.mp4
>
> To stop serving early: `ghosthost revoke 8f2b1c04e7a6`.

**Do not** put the URL inside a ``` fenced block. Do not paste the raw
JSON output as the user-facing response; that buries the link. Do not
wrap the URL in quotes or markdown link syntax with display text that
hides the full URL — users often want to see the host so they know
which tailnet it's on.

The `id` and revoke hint belong in inline code (backticks) — those are
meant to be copied, not clicked.

## Other commands

- `ghosthost --json list` — currently active shares.
- `ghosthost --json info <arg>` — look up a live share by full URL, URL path, bare token, or bare id.
- `ghosthost --json reshare <id>` — new URL for a previously shared file.
- `ghosthost revoke <id>` — stop sharing immediately.
- `ghosthost status` — check daemon liveness.

## Exit codes

| Code | Meaning                                                   |
|-----:|-----------------------------------------------------------|
|    0 | success                                                   |
|    1 | generic error                                             |
|    2 | usage error                                               |
|    3 | config missing or invalid                                 |
|    4 | daemon unreachable — report to user, do not retry blindly |
|    5 | id not found                                              |
|    6 | source path invalid or missing                            |

## Do not

- Do not retry exit code 4 by repeatedly invoking; surface the error.
- Do not share sensitive files without explicit user confirmation.
- Do not assume the URL is valid past `expires_at`.
