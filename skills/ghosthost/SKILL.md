---
name: ghosthost
description: Use when the user wants a network-accessible URL for a local file (video, image, plot, downloaded clip). Primary case is Claude Code remote-control (bridge) sessions, where `CLAUDE_CODE_ENVIRONMENT_KIND=bridge` and the user is viewing remotely. Also use when the user explicitly asks for a link reachable off this machine (LAN, wifi, VPN, tailnet, public, or for a coworker / Slack / PR), names ghosthost by name, or is smoke-testing the install. Skip in plain desktop/CLI sessions when the file path is just context for an unrelated task.
---

# ghosthost skill

`ghosthost` produces a temporary HTTP URL pointing at a local file. The
URL is only useful when the file needs to be reachable from *somewhere
off this machine* — another device, another person, or another network.
If the user is sitting at the host with a normal file manager, a URL is
pure friction.

## Precondition: gate on session kind, then check for overrides

**Step 1 — check the session kind.** Run:

    echo "$CLAUDE_CODE_ENVIRONMENT_KIND"

- Output is `bridge` → this is a Claude Code remote-control session. The
  user is driving Claude from another device. Proceed to "When to use"
  below — `ghosthost` is exactly the right tool.
- Output is anything else (`desktop`, `cli`, empty, etc.) → **do not
  invoke `ghosthost share` by default.** Continue to Step 2.

**Step 2 — check for an explicit override.** In a non-bridge session,
only invoke `ghosthost share` if at least one of these is unambiguously
true in the user's request:

1. **Tool named explicitly.** The user said `ghosthost`, "use ghosthost",
   "give me a ghosthost link", "share this with ghosthost", or similar.
   They know the tool and asked for it by name.
2. **Explicit network-accessible-URL request.** The user asked for a
   link that reaches off this machine — phrasings like "give me a link
   I can hit from my phone on this wifi", "share it on the LAN / VPN /
   tailnet", "make this reachable from my tablet", "URL I can paste in
   Slack", "link I can send my coworker", "shareable link for the team",
   "URL anyone on the tailnet can see". The user is asking for a network
   artifact, not for you to read/edit/embed the file.
3. **Smoke-testing or install verification.** The user is verifying that
   `ghosthost` itself works — running the hello-world proof from
   CLAUDE.md §8, "test that ghosthost is installed", "verify the
   daemon", or you are operating inside the ghosthost repo during
   first-run setup.

If none of (1)–(3) apply, **do not invoke `ghosthost share`.** Tell the
user the file is local and give them the absolute path so they can open
it directly. Do not infer remote-control from conversational cues alone
("I'm on my phone", "send it to me", "show it to me") — users say that
in local sessions too, and the env var is the authoritative signal for
the bridge case. The overrides above are the only carve-outs.

## When to use

Once the precondition is satisfied (bridge session, or one of the
overrides above), invoke `ghosthost share` when:

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

       ghosthost --json share <absolute-path>... [--ttl 2h] [--as <name>] [--anon]

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
