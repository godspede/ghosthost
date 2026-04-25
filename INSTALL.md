# Installing ghosthost

Install guide for `ghosthost`. Pick one of: prebuilt release binary (recommended), `go install` from source, or clone + build.

---

## Have Claude do it for you

If you already run [Claude Code](https://claude.com/claude-code) (or any similarly-capable tool-using LLM with shell access) on the machine you want to install `ghosthost` on, the fastest path is to ask it. Start a session on that machine and paste this prompt:

```text
Help me install ghosthost and its Claude skill on this machine.
Follow the install guide and operator checklist in the repo:
https://github.com/godspede/ghosthost
When you're finished, share a small hello-world text file via
ghosthost and give me the URL so I can confirm it works.
```

Claude will read `INSTALL.md`, `CLAUDE.md`, and `skills/ghosthost/SKILL.md`, pick the right install option for your OS, wire up a reachable transport (it will ask what you want to use), install the skill into your Claude skills directory, and hand you a proof-of-install URL at the end. For most readers this is faster than doing it by hand.

The rest of this file is the manual path — useful if you don't use Claude, if you want to understand what each step is doing, or if you hit something Claude can't figure out.

---

## Recommended end-to-end flow (manual)

`ghosthost` runs a small HTTP server on your machine and prints URLs that point at it. The whole system works as soon as two things are true: the binary is installed, and the machine is reachable from wherever you want to open the URLs. "Reachable" can be any of:

- **Tailscale** (easiest; what `ghosthost` ships tuned for). The daemon binds to your tailnet interface, `host` auto-populates from `tailscale status --json`, and URLs only exist on your tailnet. No firewall holes, no DNS setup. Free personal tier is enough. Install from [tailscale.com/download](https://tailscale.com/download).
- **Same LAN.** Set `bind` and `host` to the machine's LAN IP. Open the configured port on the OS firewall.
- **Another VPN** (WireGuard, Nebula, ZeroTier, etc.). Set `bind` and `host` to the VPN-interface IP or hostname. Treat the VPN as the trust boundary — tokens are the only authentication on the wire.
- **Public exposure** via a reverse proxy (Caddy, nginx, Cloudflare Tunnel) terminating TLS in front of the daemon. `host` is your public hostname, `bind` is whatever the proxy can reach. You can also skip the proxy and let `ghosthost` terminate TLS itself via `tls_cert` / `tls_key`. Either way, public exposure is at your own risk — a 128-bit token is the only gate.

Once you've picked a transport, the flow is the same:

1. **Install the binary.** Pick Option A, B, or C below, verify with `ghosthost --help`.
2. **Initialize the config.** Run any command (e.g. `ghosthost status`). First run writes a template at `%APPDATA%\ghosthost\config.toml` (Windows) or `~/.config/ghosthost/config.toml` (Linux/macOS) and exits with a friendly error. Edit `host` and `bind` for your chosen transport. If Tailscale is installed and up, the template comes pre-filled with your MagicDNS name.
3. **Share a file.**
   ```bash
   ghosthost share ./some-file.mp4
   ```
   You get a URL, an `id`, and an expiry. Open the URL on any device that can reach `host`:`port`.
4. **(Optional) HTTPS.** Point `tls_cert` and `tls_key` in the config at PEM files and `ghosthost share` starts returning `https://` URLs. Any cert/key pair works; Tailscale users can produce a browser-trusted pair in one command with `tailscale cert <your-magicdns-name>`.
5. **(Optional) Claude skill.** If you drive `ghosthost` from Claude Code, copy `skills/ghosthost/SKILL.md` into the Claude skills directory so the agent reaches for `ghosthost share` on its own. The skill is scoped to **Claude Code remote-control (bridge) sessions** — it fires automatically when Claude is launched via `claude remote-control` (detected via `CLAUDE_CODE_ENVIRONMENT_KIND=bridge`). In a plain desktop/CLI session it stays out of the way unless you explicitly name `ghosthost`, ask for a network-accessible URL, or are smoke-testing the install. Concrete per-OS commands live in [CLAUDE.md](CLAUDE.md).

Full end-to-end setup, troubleshooting, and validation checkboxes: see [CLAUDE.md](CLAUDE.md).

---

## Option A — Prebuilt release binary (recommended)

All artifacts live on the [v0.1.0 release page](https://github.com/godspede/ghosthost/releases/tag/v0.1.0). Each `.zip` contains a single `ghosthost` binary (`ghosthost.exe` on Windows) and is accompanied by a `.sig` file for cosign verification (see below).

### Linux (amd64 and arm64)

Pick the archive that matches `uname -m` — `x86_64` → `amd64`, `aarch64` → `arm64`.

```bash
# amd64
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_linux_amd64.zip
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_linux_amd64.zip.sig

# arm64 (Raspberry Pi 4/5, Ampere, Graviton, etc.)
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_linux_arm64.zip
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_linux_arm64.zip.sig
```

Extract, mark executable, and drop it somewhere on `$PATH`:

```bash
unzip ghosthost_0.1.0_linux_amd64.zip
chmod +x ghosthost
sudo install -m 0755 ghosthost /usr/local/bin/ghosthost
ghosthost --help
```

`~/.local/bin/ghosthost` is a fine non-root alternative if that directory is already on your `PATH`.

### macOS (Intel and Apple Silicon)

Apple Silicon Macs (M1/M2/M3/M4) take the `darwin_arm64` build. Older Intel Macs take `darwin_amd64`. Rosetta is not required for arm64 if you pick the native build.

```bash
# Apple Silicon
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_darwin_arm64.zip
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_darwin_arm64.zip.sig

# Intel
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_darwin_amd64.zip
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_darwin_amd64.zip.sig
```

Extract and install:

```bash
unzip ghosthost_0.1.0_darwin_arm64.zip
chmod +x ghosthost
xattr -d com.apple.quarantine ghosthost
sudo install -m 0755 ghosthost /usr/local/bin/ghosthost
ghosthost --help
```

The `xattr -d com.apple.quarantine` step is required because the binary is cosign-signed but not Apple-notarized. Gatekeeper will otherwise refuse to execute a downloaded binary with a "cannot be opened because the developer cannot be verified" dialog. As an alternative, run the binary once from Finder via right-click → Open and click through the prompt.

### Windows (amd64, PowerShell)

```powershell
Invoke-WebRequest -Uri https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_windows_amd64.zip -OutFile ghosthost_0.1.0_windows_amd64.zip
Invoke-WebRequest -Uri https://github.com/godspede/ghosthost/releases/download/v0.1.0/ghosthost_0.1.0_windows_amd64.zip.sig -OutFile ghosthost_0.1.0_windows_amd64.zip.sig
```

Unblock the archive before extracting — Windows marks files downloaded from the internet with a Mark-of-the-Web Zone Identifier (`Zone.Identifier` alternate data stream), and SmartScreen will otherwise refuse to run the extracted `.exe`:

```powershell
Unblock-File .\ghosthost_0.1.0_windows_amd64.zip
Expand-Archive .\ghosthost_0.1.0_windows_amd64.zip -DestinationPath .\ghosthost-bin
```

(Equivalent GUI path: right-click the `.zip` → Properties → check **Unblock** → OK.)

Move the binary somewhere on `PATH`. A per-user install under `%LOCALAPPDATA%` avoids needing admin:

```powershell
$dest = "$env:LOCALAPPDATA\Programs\ghosthost"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Move-Item .\ghosthost-bin\ghosthost.exe "$dest\ghosthost.exe" -Force
# Add to user PATH (once):
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$dest", "User")
# Reopen your shell, then:
ghosthost --help
```

---

## Verifying signatures (recommended for all downloads)

Release artifacts are signed with **cosign keyless**: there is no long-lived signing key, each release is signed by the repository's own OIDC identity at build time via GitHub Actions, and the signature is backed by a Sigstore transparency-log entry at Rekor. Verification checks that the signature was produced by `godspede/ghosthost`'s release workflow at the `v0.1.0` tag and by no one else.

Install cosign: see [docs.sigstore.dev/cosign/system_config/installation](https://docs.sigstore.dev/cosign/system_config/installation/).

Verify an archive (Linux amd64 shown; adjust the filename for your platform):

```bash
cosign verify-blob \
  --certificate-identity 'https://github.com/godspede/ghosthost/.github/workflows/release.yml@refs/tags/v0.1.0' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --signature ghosthost_0.1.0_linux_amd64.zip.sig \
  ghosthost_0.1.0_linux_amd64.zip
```

Success prints `Verified OK` and exits 0. Any other outcome — a non-zero exit, a mismatched certificate identity, a missing Rekor entry — means **do not run the binary**. Re-download, and if it still fails, open an issue and treat it as a supply-chain incident per `SECURITY.md`.

## Verifying checksums (optional secondary check)

Redundant with cosign if you already verified the signature, but useful as a quick integrity check or in environments without cosign.

```bash
curl -LO https://github.com/godspede/ghosthost/releases/download/v0.1.0/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

On Windows:

```powershell
Invoke-WebRequest -Uri https://github.com/godspede/ghosthost/releases/download/v0.1.0/checksums.txt -OutFile checksums.txt
$expected = (Select-String -Path checksums.txt -Pattern 'ghosthost_0.1.0_windows_amd64.zip').Line.Split(' ')[0]
$actual = (Get-FileHash .\ghosthost_0.1.0_windows_amd64.zip -Algorithm SHA256).Hash.ToLower()
if ($expected -eq $actual) { "OK" } else { "MISMATCH"; exit 1 }
```

`checksums.txt` itself is cosign-signed as `checksums.txt.sig` — verify it with the same `cosign verify-blob` incantation above if you want the checksum file itself attested.

---

## Option B — `go install` from source

```bash
go install github.com/godspede/ghosthost/cmd/ghosthost@v0.1.0
```

Requires **Go 1.25 or newer** (per `go.mod`). The version string baked into the resulting binary (`ghosthost --version`) will be `v0.1.0`.

Slower than downloading a prebuilt and depends on a working Go toolchain, but it bypasses any "is the prebuilt binary safe" question if you already trust the Go module proxy and the `godspede/ghosthost` source tree.

## Option C — Clone and build

Use this if you intend to hack on the tool.

```bash
git clone https://github.com/godspede/ghosthost.git
cd ghosthost
go build -o ghosthost ./cmd/ghosthost
./ghosthost --help
```

---

## After installing

First invocation of any command writes a template config at `%APPDATA%\ghosthost\config.toml` (Windows) or `~/.config/ghosthost/config.toml` (Linux/macOS) and exits with a friendly error pointing at the file. If Tailscale is installed and logged in, the `host` key is pre-filled from `tailscale status --json`.

Full end-to-end setup — config initialization, Tailscale wiring, first share, Claude skill install, hello-world proof-of-install, revoke, troubleshooting, validation checkboxes — lives in [CLAUDE.md](CLAUDE.md). Start there.

---

## Uninstalling

```bash
ghosthost stop          # release the daemon lockfile first
```

Then delete:

- The binary (`/usr/local/bin/ghosthost`, `~/.local/bin/ghosthost`, or `%LOCALAPPDATA%\Programs\ghosthost\ghosthost.exe`).
- The config directory: `%APPDATA%\ghosthost\` (Windows) or `~/.config/ghosthost/` (Linux/macOS).
- The data directory: `%LOCALAPPDATA%\ghosthost\` (Windows) or `~/.local/share/ghosthost/` (Linux/macOS) — or wherever `data_dir` in `config.toml` points. This is where `history.jsonl` and the daemon lockfile live.

Running `ghosthost stop` before deletion avoids orphaning a running daemon that's still holding the lockfile open.

## Upgrading

```bash
ghosthost stop
# replace the binary using the same steps as a fresh install
ghosthost status
```

Config and `history.jsonl` carry over unchanged. Within a major version (v0.x → v0.y), JSON schemas are append-only per the stability pledge in [README.md](README.md#json-output-stability), so existing scripts and the Claude skill keep working across minor upgrades.

---

## Troubleshooting

- **"Windows protected your PC" / SmartScreen blocked the exe.** The downloaded archive kept its Mark-of-the-Web. Right-click the `.zip` → Properties → check **Unblock**, or run `Unblock-File .\ghosthost_0.1.0_windows_amd64.zip` before extracting. Re-extract afterward.
- **`zsh: command not found: ghosthost` / `'ghosthost' is not recognized as an internal or external command`.** The binary isn't on `PATH`. Check with `which ghosthost` (Linux/macOS) or `where.exe ghosthost` (Windows). Either move the binary into a directory already on `PATH` or extend `PATH` to include where you put it.
- **`permission denied` running `./ghosthost` on Linux or macOS.** The executable bit didn't make it through the unzip. `chmod +x ghosthost`.
- **Anything else** — daemon won't start, Tailscale detection misbehaves, shares 404, admin API auth fails — see the troubleshooting section of [CLAUDE.md](CLAUDE.md). For suspected vulnerabilities, use the disclosure channel in [SECURITY.md](SECURITY.md); don't open a public issue.
