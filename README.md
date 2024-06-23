# fen

[![Go Report Card](https://goreportcard.com/badge/github.com/kivattt/fen)](https://goreportcard.com/report/github.com/kivattt/fen)

fen is a terminal file manager inspired by [ranger](https://github.com/ranger/ranger)\
Works for Linux, macOS, FreeBSD and Windows

<p float="left">
<img src="screenshots/linux.png" alt="fen running on Linux, with the file preview rainbow.lua" width="48%">
<img src="screenshots/macos.png" alt="fen running on macOS, showing the no-write feature" width="50%">
<img src="screenshots/freebsd.png" alt="fen running on FreeBSD, showing the root file system" width="50%">
<img src="screenshots/windows.png" alt="fen running on Windows, showing the obsolete file properties window" width="48%">
</p>

# Try it out now!
```
go run github.com/kivattt/fen@latest
```

# Installing on Linux/FreeBSD
Download the latest version in the [Releases](https://github.com/kivattt/fen/releases) page, and put it inside `/usr/local/bin`

Alternatively:
```
sudo -i GOBIN=/usr/local/bin go install github.com/kivattt/fen@latest
```

# Building
```
go build
./fen
```

# Controls
Arrow keys, hjkl or scrollwheel to navigate (Enter goes right), Escape key to cancel an action

`?` or `F1` Toggle help menu\
`Ctrl + Space` or `Ctrl + n` Open file(s) with specific program\
`Home` or `g` to go to the top\
`End` or `G` to go to the bottom\
`M` Go to the middle\
`H` Go to the top of the screen\
`L` Go to the bottom of the screen\
`q` Quit\
`Del` Delete file(s)\
`y` Copy file(s)\
`d` Cut file(s)\
`p` Paste file(s)\
`/` or `Ctrl + f` Search\
` ` Select files\
`A` Flip selection in folder (select all files)\
`D` Deselect all, and un-yank\
`a` Rename a file\
`z` or `Backspace` Toggle hidden files\
`V` Start selecting by moving\
`n` Create a new file\
`N` Create a new folder

# Configuration
You can find a complete example config in the [fenrc.json](fenrc.json) file

Linux/FreeBSD: `~/.config/fen/fenrc.json` or `$XDG_CONFIG_HOME/fen/fenrc.json` if `$XDG_CONFIG_HOME` set\
macOS: `$HOME/Library/Application Support/fen/fenrc.json`\
Windows: `%AppData%\Roaming\fen\fenrc.json`

You can specify a different config file path with the `--config` flag

The `"open-with"` file matching starts from the top, so you can have something like this at the end of the list to catch anything not previously matched:
```json
{
    "programs": ["vim -p"],
    "match": ["*"]
}
```

You can use "do-not-match" in conjunction with "match":
```json
{
    "programs": ["notepad"],
    "match": ["*"],
    "do-not-match": ["*.exe"]
}
```

Programs in `"programs"` do not expand tildes like `"~/some/file.sh"`. You have to specify an absolute path.\
However, `FEN_CONFIG_PATH` will be expanded to the config path, see: [Configuration](#Configuration)\
If you want to use a shell script in `"programs"`, it has to have a shebang or you need to explicitly invoke the appropriate shell like `"bash /some/file.sh"`\
Note: Programs will be started in the working directory you're inside in fen

# File previews
fen does not (yet!) have file previews by default\
For file previews with programs like `cat` or `head`, you can add something like this to your fenrc.json:
```json
"preview-with": [
    {
        "programs": ["head -n 100"],
        "match": ["*"]
    }
]
```

For something cross-platform, file previews can also be a [lua script](lua-file-preview-examples/basic.lua). You can use them by setting "script" in "preview-with":
```json
"preview-with": [
    {
        "script": "basic.lua",
        "match": ["*"]
    }
]
```
If "script" is set, "programs" will be ignored in the same "preview-with" entry.\
"script" is not a list like "programs" is, because we want to see syntax errors when writing lua code instead of falling back to anything.\
The "script" key has to be an absolute path e.g. `"/home/user/my-script.lua"`, however `FEN_CONFIG_PATH` will be expanded to the config path, see: [Configuration](#Configuration)

For backwards compatibility reasons, a "script" path which isn't an absolute path is valid and will be automatically prepended with the config path (same as `FEN_CONFIG_PATH`)

# Writing file preview scripts with Lua
Do not use `print()`, it outputs to stdout which doesn't work well within fen.\
You can find examples in [lua-file-preview-examples](lua-file-preview-examples)

### Available functions:
`fen:Print(text, x, y, maxWidth, alignment, color)` Print text at the given x/y position. x=0, y=0 is the top left corner of the file preview area and limited to the file preview area only [Go doc](https://pkg.go.dev/github.com/rivo/tview#Print)\
`fen:PrintSimple(text, x, y)` Same as above, with default color and alignment and no maxWidth [Go doc](https://pkg.go.dev/github.com/rivo/tview#PrintSimple)\
`fen:Escape(text)` Escape style tags [Go doc](https://pkg.go.dev/github.com/rivo/tview#Escape)\
`fen:TranslateANSI(text)` Turn ANSI into style tags [Go doc](https://pkg.go.dev/github.com/rivo/tview#TranslateANSI)\
`fen:NewRGBColor(r, g, b)` [Go doc](https://pkg.go.dev/github.com/gdamore/tcell/v2#NewRGBColor)\
`fen:ColorToString(color)` [Go doc](https://pkg.go.dev/github.com/gdamore/tcell/v2#Color.String)\
`fen:RuntimeOS()` The OS fen is running in [Go doc](https://pkg.go.dev/runtime#pkg-constants)

Notes about `fen:Print()` and `fen:PrintSimple()`:\
Newlines will not show up, and do nothing. You will have to manually call it multiple times, increasing y.\
Tabs are replaced with 4 spaces so they are visible

### Available variables:
`fen.SelectedFile` Currently selected file absolute file path to preview\
`fen.Width` Width of the file preview area\
`fen.Height` Height of the file preview area

# Known issues
- On FreeBSD, when the disk is full, fen may erroneously show a very large amount of disk space available (like `18.446 EB free`), when in reality there is no available space
- Deleting files sometimes doesn't work on Windows
- Setting a boolean command-line flag to false, e.g. `--no-write=false` has no effect, and the configuration file value will be prioritized. You can disable loading the config file by giving a bogus filename: `--config=aaaaa`
- `go test` doesn't work on Windows
- The color for audio files is invisible in the default Windows Powershell colors, but not cmd or Windows Terminal

See [TODO.md](TODO.md) for other issues and possible future features, roughly sorted by priority
