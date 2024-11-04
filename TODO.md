## Source code comments with "PLUGINS:" in them are for things we have to change to make Lua plugins a possibility in the future

## TODOs, vaguely sorted by priority

- Scrollable search history
- Better scrolling
- It sometimes exits badly, stuff is left on screen ever since async file operations were added
- Interactive file operations log (with undo when applicable)
- Make file previews async
- Changing owner/group, chmod inside fen (probably not, since you can do it with open-with)
- Make draw functions for top bar / bottom bar scriptable with lua
- Global selection (selection stored in a file under UserCacheDir ?)
- Ctrl+Shift+n, Ctrl+Shift+n search by content, search by path name like telescope
- Check if [dragon](https://github.com/mwh/dragon) works, maybe just make my own built into fen with some gtk wrapper? (bad idea lol)
- Show current folder size beside disk size?
- A sort of --no-unicode option, to print the character codes instead of fancy unicode characters
- Configuration: Matching based on file permission flags (like executables)? (Maybe not now that we have open Lua scripts
- Configurable colors / respect LS\_COLORS?
- Fix a crash (fen hanging) on something like `/proc/.../oom_score_adj`
- Fix the bottom bar sometimes not showing info on files inside `/proc/.../map_files`
- Replace github.com/otiai10/copy with my own recursive folder copying
- Cache folder size (file count)
- Warning message or enable hidden files when creating a new hidden file/folder
- Allow creating new files/folders with absolute paths (use fen.GoPath())
- A sort of "back arrow" key for going to the last folder we were in
- Add right pane disappearing when no preview/folder?
- Remove local tracked git repository when .git folder not found anymore
- topbar.go: Show left part of path also with invisible unicode symbols as codepoints highlighted, and also show symlinks in blue like ranger
- Configurable filespane proportions

- Abstract away this common pattern:
```go
rel, err := filepath.Rel(basePath, path)
if err != nil {
	return
}

// If it would end up going to the left, return
if strings.HasPrefix(rel, "..") {
	return
}
```

- File list mode ("flattened mode", "flattened folder view" ?)
  - Recursive directory iterator in separate thread updating the entries
  - Color change in UI, like (red? maybe something friendlier...) background for the topbar
  - Probably some text letting you know file list mode is enabled
  - Make middlePane take up the entire screen?
  - Show more file info in filespane drawing

- System-wide configuration file instructions
- Installation instructions for Windows in the README
- Allow spaces in "programs" path in config with `\ `? (Maybe not, this might be annoying on Windows), maybe add os package ExpandEnv to allow using environment variables in "programs"
- .deb file in Releases
- Allow opening images with 'feh', fix it not breaking fen, 'xviewer' can also break fen rarely
- Make the "open with" modal a selectable list with tab/shift+tab controls aswell as arrow keys, would replace inputfield placeholder and reset input text to blank
- Configurable keybindings
- Configurable custom themes by changing `tview.Styles`
- Disallow recursive copies or whatever
- Fix `history_test.go` for Windows paths
- Fix green color for all executables (the current bitmask check doesn't work for everything)
- Fix invisibility near root dir (easy to see on Android with Termux)
- `H` and `L` controls feel weird because the screen scrolls in a specific way instead of just setting the cursor to the bottom of the screen like the behaviour in vim
- Optional xdg trash specification? https://specifications.freedesktop.org/trash-spec/trashspec-latest.html
