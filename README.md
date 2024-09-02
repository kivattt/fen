# fen

[![Go Report Card](https://goreportcard.com/badge/github.com/kivattt/fen)](https://goreportcard.com/report/github.com/kivattt/fen)

fen is a terminal file manager inspired by [ranger](https://github.com/ranger/ranger)\
Works for Linux, macOS, FreeBSD and Windows

<p float="left">
<img src="screenshots/linux.png" alt="fen running on Linux, with the file preview script rainbow.lua" width="48%">
<img src="screenshots/macos.png" alt="fen running on macOS, showing the no-write feature" width="50%">
<img src="screenshots/freebsd.png" alt="fen running on FreeBSD, showing the root file system" width="50%">
<img src="screenshots/windows.png" alt="fen running on Windows, showing the open-with modal" width="48%">
</p>

# Try it out now!
```
git clone https://github.com/kivattt/fen
cd fen
go build
./fen
```

# Installing on Linux/FreeBSD
Download the latest version in the [Releases](https://github.com/kivattt/fen/releases) page, and put it inside `/usr/local/bin`

Alternatively:
```
sudo -i GOBIN=/usr/local/bin go install github.com/kivattt/fen@latest
```

# Controls
Arrow keys, hjkl, mouse click or scrollwheel to navigate (Enter goes right), Escape key to cancel an action

`?` or `F1` Toggle help menu\
`q` Quit fen\
`z` or `Backspace` Toggle hidden files\
`Ctrl + Space` or `Ctrl + n` Open file(s) with specific program\
`Home` or `g` to go to the top\
`End` or `G` to go to the bottom\
`M` Go to the middle\
`Page Up` / `Page Down` Scroll up/down an entire page\
`H` Go to the top of the screen\
`L` Go to the bottom of the screen\
`Del` or `x` Delete file(s)\
`y` Copy file(s)\
`d` Cut file(s)\
`p` Paste file(s)\
`/` or `Ctrl + f` Search\
`c` Goto folder\
`Space` Select files\
`A` Flip selection in folder (select all files)\
`D` Deselect all, and un-yank\
`a` Rename a file\
`V` Start selecting by moving\
`n` Create a new file\
`N` Create a new folder\
`F5` Sync the screen, fixes broken output that can be caused by running a command, or filenames with certain Unicode characters
`1-9` To enter a Bookmark

# Configuration
You can find a complete default config with extra examples in the [config.lua](config.lua) file\
For a full config folder example, see [my personal config](https://github.com/kivattt/dotfiles/blob/main/.config/fen/config.lua)

Linux/FreeBSD: `~/.config/fen/config.lua` or `$XDG_CONFIG_HOME/fen/config.lua` if `$XDG_CONFIG_HOME` set\
macOS: `$HOME/Library/Application Support/fen/config.lua`\
Windows: `%AppData%\Roaming\fen\config.lua`

You can specify a different config file with the `--config` flag

# File previews
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

# Changing directory
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

# Lua scripting
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

# Known issues
- fen may crash in the middle of deleting files due to a data race, most commonly when deleting a lot of files (like 4000)
- File previews are ran synchronously, which means they slow down fen
- fen intentionally does not handle Unicode "grapheme clusters" (like chinese text) in filenames correctly for performance reasons. You need to manually build fen with the replace directive for my [tcell fork](https://github.com/kivattt/tcell-naively-faster) in the go.mod file removed to show them correctly
- Symlinks have no special distinction, a folder symlink will appear like a normal folder
- On FreeBSD, when the disk is full, fen may erroneously show a very large amount of disk space available (like `18.446 EB free`), when in reality there is no available space
- Deleting files sometimes doesn't work on Windows (due to files being open in another program?)
- `go test` doesn't work on Windows
- The color for audio files is invisible in the default Windows Powershell colors, but not cmd or Windows Terminal

See [TODO.md](TODO.md) for other issues and possible future features, roughly sorted by priority
