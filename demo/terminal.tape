# demo/terminal.tape — regenerate with `vhs demo/terminal.tape`
# Produces docs/demo-terminal.gif showing a scripted `ghosthost share`
# invocation. Uses demo/bin/ghosthost (a stand-in, not the real CLI) so
# the recording environment contributes no real state, hostnames, or
# file paths to the output.
#
# Install vhs: https://github.com/charmbracelet/vhs

Output docs/demo-terminal.gif

Require bash

Set Shell     "bash"
Set FontSize  16
Set Width     960
Set Height    320
Set Theme     "Dracula"
Set TypingSpeed 40ms

# Stage a clean shell without leaking real PATH or prompt hints.
Hide
Type `export PS1="$ "`
Enter
Type `PATH="$PWD/demo/bin:$PATH"`
Enter
Type `clear`
Enter
Sleep 200ms
Show

Sleep 600ms
Type `ghosthost share ./renders/out.mp4`
Sleep 250ms
Enter
Sleep 1800ms
