# CLAUDE.md — ghosthost operator checklist

## Purpose

This file is an end-to-end setup and smoke-test checklist for `ghosthost`, written for a coding LLM (or human operator) bringing the tool up on a fresh machine. Work top to bottom: install, configure, validate, wire into Claude, then exercise the alternative-transport and troubleshooting sections only if needed.

## Prerequisites

- Go 1.25 or newer (`go version`).
- A network path from the target viewing device to the host. Tailscale is the designed-for default; alternatives are the same LAN, another VPN (WireGuard / Nebula / ZeroTier), or a public reverse proxy / tunnel. Everything below assumes Tailscale unless the "Alternative transports" section is invoked.

## 1. Install the binary

```bash
go install github.com/godspede/ghosthost/cmd/ghosthost@latest
```

Or build from a local checkout:

```bash
go build -o ghosthost ./cmd/ghosthost
```

Or download a prebuilt release from the Releases page (once available).

Confirm:

```bash
ghosthost --help
```

## 2. Tailscale (recommended)

Install Tailscale from <https://tailscale.com/download>, authenticate the host (`tailscale up`), and verify:

```bash
tailscale status
```

On first run, `ghosthost` shells out to `tailscale status --json` and pre-fills `host` in the config template with the host's MagicDNS name.

## 3. Initialize config

Running any command with no config in place writes a template and exits with a friendly error:

```bash
ghosthost share anything
```

Default config paths:

- Windows: `%APPDATA%\ghosthost\config.toml`
- Linux/macOS: `~/.config/ghosthost/config.toml`

Override with `--config <path>` on any command. Inspect and edit the template:

```toml
host          = "homepc.tail-4a9c2e.ts.net"
bind          = "tailscale"
port          = 8750
admin_port    = 8751
data_dir      = "C:\\Users\\you\\AppData\\Local\\ghosthost"
default_ttl   = "24h"
idle_shutdown = "30m"
```

Optional HTTPS: set `tls_cert` and `tls_key` to PEM file paths to make the public server serve TLS and emit `https://` URLs. See the "HTTPS (optional)" section in the README for the details, including the `tailscale cert` recipe for browser-trusted certs.

## 4. First share — sanity check

```bash
ghosthost share ./some-file.png
```

Expect one line: the URL. (Pass `--verbose` to also see the id and expiry, or run `ghosthost info <url>` afterwards.) From another device on the tailnet:

```bash
curl -I "https://homepc.tail-4a9c2e.ts.net:8750/s/<token>/some-file.png"
```

A `200 OK` with a plausible `Content-Type` means the data plane is working. If testing purely locally, set `bind = "127.0.0.1"` and `host = "127.0.0.1"` and `curl` from the same machine.

## 5. Inspect state

```bash
ghosthost list              # active shares
ghosthost history --limit 20
ghosthost status            # daemon liveness
```

Add `--json` to any command for machine-readable output.

## 6. Revoke and stop

```bash
ghosthost revoke <id>
curl -I "<same-url>"        # expect 404
ghosthost stop              # daemon exits; next command auto-spawns it
```

## 7. Install the Claude skill

The skill is installed **per-user** (globally), so it is available in every Claude session on this machine regardless of which workspace or project is open.

Copy `skills/ghosthost/SKILL.md` into the user-level Claude skills directory:

```bash
# macOS / Linux
mkdir -p ~/.claude/skills/ghosthost
cp skills/ghosthost/SKILL.md ~/.claude/skills/ghosthost/SKILL.md
```

```powershell
# Windows (PowerShell)
New-Item -ItemType Directory -Force "$env:USERPROFILE\.claude\skills\ghosthost"
Copy-Item skills\ghosthost\SKILL.md "$env:USERPROFILE\.claude\skills\ghosthost\SKILL.md"
```

**Status on this machine:** already installed at `%USERPROFILE%\.claude\skills\ghosthost\SKILL.md`.

Restart Claude after copying.

### When the skill fires

The skill is **scoped to Claude Code remote-control (bridge) sessions**
— specifically, sessions where the environment variable
`CLAUDE_CODE_ENVIRONMENT_KIND` equals `bridge`. That is the variable
Claude Code sets when you launch it via `claude remote-control`. The
rationale is simple: `ghosthost` only earns its keep when the user is
*not* sitting at the host with a normal file manager. In a local
desktop or plain-CLI session, generating a tokenized URL for a file the
user can already double-click is friction, not magic.

In a **bridge session**, Claude reaches for `ghosthost share` on its
own when you ask to see, host, or share a local file.

In a **non-bridge session** (desktop, CLI, or anything else), Claude
leaves the skill alone *unless one of these explicit overrides applies*:

1. **You name the tool.** "Use ghosthost", "give me a ghosthost link",
   "share with ghosthost".
2. **You ask for a network-accessible URL.** "Give me a link my phone
   on this wifi can hit", "share it on the LAN / VPN / tailnet", "URL
   I can paste in Slack", "link I can send my coworker", "shareable
   link for the team". The trigger is *network-link intent*, regardless
   of who the consumer is.
3. **You're smoke-testing the install.** Running §8's hello-world,
   verifying the daemon, or working inside the ghosthost repo at
   first-run setup.

Conversational cues like "I'm on my phone" or "send it to me" are
**not** overrides on their own — local users say those things too. If
you really do want a URL in a desktop session, the cleanest path is to
say "use ghosthost" or describe the network-link use case explicitly.

## 8. Hello-world proof-of-install

This is the canonical end-to-end proof that a fresh install works. Use it as a manual acceptance test and as the thing an LLM operator reports back after setup.

```bash
# Pick a stable absolute path outside any temp dir so it isn't GC'd mid-test.
HELLO_PATH="$HOME/ghosthost-hello.txt"        # macOS/Linux
# HELLO_PATH="$env:USERPROFILE\ghosthost-hello.txt"   # PowerShell on Windows

printf 'hello from ghosthost\n' > "$HELLO_PATH"
ghosthost --json share "$HELLO_PATH"
```

Expected output (JSON, one line): a JSON array with a single element containing `schema_version`, `id`, `token`, `url`, and `expires_at`.

**Human-mode output (no `--json`):** `ghosthost share <path>` prints one line — the URL. Nothing else. Pass `--verbose` for the old rich block, or run `ghosthost info <url-or-id>` to retrieve metadata for an active share.

Report the result back to the user in a format that keeps the URL **clickable in the chat UI**. Most chat UIs (including Claude Code) auto-link bare URLs in prose but NOT inside fenced code blocks. So:

- The **URL** goes on its own line, **outside** any ``` fence, as a bare `http://...` string.
- The **local path** and **id** go in inline code (single backticks) since they're meant to be copied.
- The **expiry** is plain text.

Render it like this (the literal template — adapt the values):

> Installed and shared.
>
> **Local file:** `C:\Users\you\ghosthost-hello.txt`
>
> http://homepc.tail-4a9c2e.ts.net:8750/t/k3n.../ghosthost-hello.txt
>
> **ID:** `8f2b1c04e7a6` — stop early with `ghosthost revoke 8f2b1c04e7a6`
>
> **Expires:** 2026-04-20T14:32:08Z
>
> Local self-check: HTTP 200, body match.

**Do not** put the URL inside a ``` fence. **Do not** paste the raw JSON as the user-facing response. The whole point of this tool is handing the user a link they can tap — if the link isn't tappable, the proof has failed.

Verification: open the URL in a browser on another device (or `curl` it locally). You should see `hello from ghosthost`. That's the install confirmed working end to end — binary, daemon, config, network path, and Tailscale (or whatever transport you picked) are all wired up.

**Multiple files:** `ghosthost share a.png b.png c.png` creates three shares in one invocation and prints one URL per line. Pass `--anon` to replace each URL's filename segment with a random slug while preserving the extension.

Leave the share active for manual inspection, then `ghosthost revoke <id>` when done.

## Alternative transports

For operators not using Tailscale:

- **Same LAN.** Set `bind` to the host's LAN IP (e.g. `"192.168.1.42"`) and `host` to the same IP or an mDNS name (`homepc.local`). Ensure the OS firewall permits inbound on `port` and `admin_port`'s loopback binding is unaffected.
- **Other VPN** (WireGuard, Nebula, ZeroTier). Set `bind` and `host` to the VPN interface IP or hostname. Treat the VPN as the trust boundary.
- **Public exposure.** Run a reverse proxy (Caddy, nginx, Cloudflare Tunnel) terminating TLS in front of the daemon. Set `host` to the public hostname and `bind` to an interface the proxy can reach. Tokens are the only authentication at this point — public exposure is at your own risk. As an alternative to a reverse proxy, the daemon itself can terminate TLS via the `tls_cert` / `tls_key` config keys (see the HTTPS section in the README) — useful if you want browser-trusted HTTPS on your tailnet without running a separate proxy.

## Troubleshooting

- **Exit code 4, "daemon unreachable."** Run `ghosthost status`. Check that `port` and `admin_port` are free and not blocked by a firewall. Inspect `<data_dir>/daemon.log` (JSON, `log/slog` format).
- **"bind=tailscale requires working tailscale."** Either start Tailscale (`tailscale up`) or change `bind` to an explicit IP in the config.
- **Can't find the config.** Windows: `%APPDATA%\ghosthost\config.toml`. Linux/macOS: `~/.config/ghosthost/config.toml`. Or pass `--config <path>`.
- **Daemon log.** `<data_dir>/daemon.log`. One JSON object per line.

## Validation checklist

- [ ] `go version` reports 1.22 or newer.
- [ ] `ghosthost --help` runs.
- [ ] `tailscale status` shows the host online (if using Tailscale).
- [ ] Config file exists at the expected path; `host` and `bind` are set.
- [ ] `ghosthost share <path>` prints one line: the URL. (`--verbose` adds id + expiry; `--json` returns a JSON array — one element per file.)
- [ ] `curl -I <url>` from a second device returns `200 OK`.
- [ ] `ghosthost list` shows the share; `ghosthost history` shows the creation event.
- [ ] `ghosthost revoke <id>` makes the URL return `404`.
- [ ] `ghosthost stop` exits cleanly; next `ghosthost status` auto-spawns the daemon.
- [ ] `skills/ghosthost/SKILL.md` is installed **per-user** at `%USERPROFILE%\.claude\skills\ghosthost\SKILL.md` (macOS/Linux: `~/.claude/skills/ghosthost/SKILL.md`) and Claude has been restarted.
- [ ] **Hello-world proof-of-install (§8):** a short text file is shared, the URL opens in a browser on another device and shows the file contents, and the `LOCAL_PATH` + `URL` pair has been recorded.
