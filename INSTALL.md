# Installing ghosthost

Install guide for `ghosthost`. Pick one of: prebuilt release binary (recommended), `go install` from source, or clone + build.

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
