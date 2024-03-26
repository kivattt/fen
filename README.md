# fen
fen is a terminal file manager based on [ranger](https://github.com/ranger/ranger)\
**Don't use this program! It's a work-in-progress and you could end up accidentally deleting files**

# Running
```
go build
./kivaranger
```

Arrow keys, hjkl or scrollwheel to navigate (Enter goes right) \
`Home` or `g` to go to the top \
`End` or `G` to go to the bottom \
`M` Go to the middle \
`q` Quit \
`Del` Delete file(s) \
` ` Select files \
`A` Flip selection in folder (select all files) \
`D` Deselect all, and un-yank \
`a` Rename a file \
`z` Toggle hidden files \
`V` Start selecting by moving \
`n` Create a new file (touch) \
`N` Create a new folder (mkdir) \
`?` Toggle file properties window

# Known issues
- Doesn't show like, root folder stuff
- `go test` doesn't work on Windows
