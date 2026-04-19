# demo/

Assets and scripts for the README demo GIFs.

## Regenerating the terminal GIF

The terminal half of the demo is produced by [charmbracelet/vhs](https://github.com/charmbracelet/vhs):

```bash
vhs demo/terminal.tape
```

Output: `docs/demo-terminal.gif`.

The tape uses `demo/bin/ghosthost`, a small bash stand-in that prints pre-baked
output. The recording has no dependency on the real binary, the real daemon,
or any real hostname / path — so regeneration is reproducible on any machine
with `vhs` installed and leaks nothing about the recorder's environment.

Edit `demo/bin/ghosthost` if the CLI's output shape changes, then re-run `vhs`.

## Phone recording

The phone half is a manual capture (see the README's top block for what
should appear). Record against a disposable Tailscale identity or blur the
URL bar in post if you prefer. Save the result at `docs/demo-phone.mp4` or
`docs/demo-phone.gif` and update the README embed.
