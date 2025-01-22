# fen

[![Go Report Card](https://goreportcard.com/badge/github.com/kivattt/fen)](https://goreportcard.com/report/github.com/kivattt/fen)

fen is a terminal file manager inspired by [ranger](https://github.com/ranger/ranger)\
Works for Linux, macOS, FreeBSD and Windows

Warning! There are race conditions which make fen unsuitable for copying large amounts of individual files. Use something like [ranger](https://github.com/ranger/ranger) for this purpose instead!

<p float="left">
<img src="screenshots/linux.png" alt="fen running on Linux, with the file preview script rainbow.lua" width="48%">
<img src="screenshots/macos.png" alt="fen running on macOS, showing the no-write feature" width="50%">
<img src="screenshots/freebsd.png" alt="fen running on FreeBSD, showing the root file system" width="50%">
<img src="screenshots/windows.png" alt="fen running on Windows, showing the open-with modal" width="48%">
</p>

## Installing
### Prebuilt binaries
Download and run the latest version in the [Releases](https://github.com/kivattt/fen/releases) page

Add it to your path environment variable, or (on Linux/FreeBSD) place the executable in `/usr/local/bin`

### Building from source
This requires Go 1.21.5 or above ([install Go](https://go.dev/dl/))
```
git clone https://github.com/kivattt/fen
cd fen
go build
./fen # fen.exe on Windows
```

## Controls
Arrow keys, hjkl, mouse click or scrollwheel to navigate (Enter goes right), Escape key to cancel an action

<kbd>?</kbd> or <kbd>F1</kbd> Toggle help menu\
<kbd>F2</kbd> Show libraries used in fen\
<kbd>q</kbd> Quit fen\
<kbd>o</kbd> Options\
<kbd>z</kbd> or <kbd>Backspace</kbd> Toggle hidden files\
<kbd>Ctrl + Space</kbd> or <kbd>Ctrl + n</kbd> Open file(s) with specific program\
<kbd>!</kbd> Run system shell command (cmd on Windows)\
<kbd>Home</kbd> or <kbd>g</kbd> Go to the top\
<kbd>End</kbd> or <kbd>G</kbd> Go to the bottom\
<kbd>Ctrl + Left arrow</kbd> Go to the root folder (or current Git repository if `fen.git_status=true`)\
<kbd>Ctrl + Right arrow</kbd> Go to the path furthest down in history, follow a symlink or go to the first changed file if `fen.git_status=true`\
<kbd>M</kbd> Go to the middle\
<kbd>Page Up</kbd> / <kbd>Page Down</kbd> Scroll up/down an entire page\
<kbd>H</kbd> Go to the top of the screen\
<kbd>L</kbd> Go to the bottom of the screen\
<kbd>Del</kbd> or <kbd>x</kbd> Delete file(s)\
<kbd>y</kbd> Copy file(s)\
<kbd>d</kbd> Cut file(s)\
<kbd>p</kbd> Paste file(s)\
<kbd>/</kbd> or <kbd>Ctrl + f</kbd> Search\
<kbd>c</kbd> Goto path\
<kbd>Space</kbd> Select files\
<kbd>A</kbd> Flip selection in folder (select all files)\
<kbd>D</kbd> Deselect all, press again to un-yank\
<kbd>a</kbd> Rename a file\
<kbd>b</kbd> Bulk-rename (rename in editor)\
<kbd>V</kbd> Start selecting by moving\
<kbd>n</kbd> Create a new file\
<kbd>N</kbd> Create a new folder\
<kbd>F5</kbd> Refreshes files, syncs the screen (fixes broken output), refreshes git status when `fen.git_status=true`\
<kbd>0-9</kbd> Go to a configured bookmark

## Configuration
You can find a complete default config with extra examples in the [config.lua](config.lua) file\
For a full config folder example, see [my personal config](https://github.com/kivattt/dotfiles/blob/main/.config/fen/config.lua)

Linux/FreeBSD: `~/.config/fen/config.lua` or `$XDG_CONFIG_HOME/fen/config.lua` if `$XDG_CONFIG_HOME` set\
macOS: `$HOME/Library/Application Support/fen/config.lua`\
Windows: `%AppData%\Roaming\fen\config.lua`

You can specify a different config file with the `--config` flag

Left-clicking to copy the selected path on Linux/FreeBSD requires `xclip` to be installed

## File previews
fen does not (yet!) have file previews by default\
For file previews with programs like `cat` or `head`, you can add something like this to your config.lua:
```lua
fen.preview = {
    {
        program = {"head -n 100"},
        match = {"*"}
    }
}
```

For something cross-platform, file previews can also be a [lua script](lua-file-preview-examples/basic.lua).
```lua
fen.preview = {
    {
        script = fen.config_path.."basic.lua",
        match = {"*"}
    }
}
```
If "script" is set, "program" will be ignored in the same preview entry.\
"script" can not be a list like "program" can, because we want to see syntax errors when writing lua code instead of falling back to anything.\
The "script" key has to be an absolute file path

## Changing directory
You can change the current working directory to the one in fen on exit:
```bash
cd $(fen --print-folder-on-exit)
```

You can alias fen to do this every time you open it by adding this to your `.bashrc`:
```bash
cd_fen() {
    cd $(fen --print-folder-on-exit)
}
alias fen=cd_fen
```
NOTE: Using this alias will break command-line arguments, like `fen -v` since the output will be passed to `cd`.

<details>
<summary><h2>Lua scripting (click to expand)</h2></summary>

fen uses [gopher-lua](https://github.com/yuin/gopher-lua) as its Lua runtime.

## Writing file preview scripts with Lua
Do not use `print()`, it outputs to stdout which doesn't work well within file previews.\
You can find examples in [lua-file-preview-examples](lua-file-preview-examples)

File preview scripts are separate from config.lua, don't expect any direct overlap in the API

### Available variables:
`fen.SelectedFile` Currently selected file absolute file path to preview\
`fen.Width` Width of the file preview area\
`fen.Height` Height of the file preview area

### Available functions:
`fen:Print(text, x, y, maxWidth, alignment, color) returns amount of characters on screen printed` Print text at the given x/y position. x=0, y=0 is the top left corner of the file preview area and limited to the file preview area only [Go doc](https://pkg.go.dev/github.com/rivo/tview#Print)\
`fen:PrintSimple(text, x, y) returns amount of characters on screen printed` Same as above, with default color and alignment and no maxWidth [Go doc](https://pkg.go.dev/github.com/rivo/tview#PrintSimple)\
`fen:Escape(text)` Escape style tags [Go doc](https://pkg.go.dev/github.com/rivo/tview#Escape)\
`fen:TranslateANSI(text)` Turn ANSI into style tags [Go doc](https://pkg.go.dev/github.com/rivo/tview#TranslateANSI)\
`fen:NewRGBColor(r, g, b)` [Go doc](https://pkg.go.dev/github.com/gdamore/tcell/v2#NewRGBColor)\
`fen:ColorToString(color)` (Since v1.1.2) [Go doc](https://pkg.go.dev/github.com/gdamore/tcell/v2#Color.String)\
`fen:RuntimeOS()` (Since v1.1.3) The OS fen is running in [Go doc](https://pkg.go.dev/runtime#pkg-constants)\
`fen:Version()` (Since v1.2.3) fen version string

**Notes about `fen:Print()` and `fen:PrintSimple()`:**\
Newlines will not show up, and do nothing. You will have to manually call it multiple times, increasing y.\
Tabs are replaced with 4 spaces so they are visible

## Writing file open scripts with Lua (Since v1.3.0)
You can find examples in [lua-file-open-examples](lua-file-open-examples)

### Available variables:
`fen.SelectedFiles` List of selected files to open\
`fen.ConfigPath` Same as `fen.config_path` from config.lua\
`fen.RuntimeOS` The OS fen is running in [Go doc](https://pkg.go.dev/runtime#pkg-constants)\
`fen.Version` fen version string
</details>

## Known issues
- fen may crash in the middle of deleting files due to a data race, most commonly when deleting a lot of files (like 4000)
- File previews are ran synchronously, which means they slow down fen
- fen intentionally does not handle Unicode "grapheme clusters" (like chinese text) in filenames correctly for performance reasons. You need to manually build fen with the replace directive for my [tcell fork](https://github.com/kivattt/tcell-naively-faster) in the go.mod file removed to show them correctly
- On FreeBSD, when the disk is full, fen may erroneously show a very large amount of disk space available (like `18.446 EB free`), when in reality there is no available space
- `go test` doesn't work on Windows
- The color for audio files is invisible in the default Windows Powershell colors, but not cmd or Windows Terminal
- Bulk-renaming a .git folder on Windows hangs fen forever

See [TODO.md](TODO.md) for other issues and possible future features, roughly sorted by priority
