This folder contains just enough boilerplate to run the "Search filenames" popup as a standalone application.\
Enough to test for data races with `go build -race`.\
Since fen has data races elsewhere, I need this to be able to look for data races only in the "Search filenames" feature of the program.

To look for race conditions in this folder:
```bash
cp ../searchfilenames.go .
go build -race
./main

# Press 'f' and try out things.
# When you're done, press Escape and then Q to quit the program.
# If it says something like "Detected ... data race(s)" there is a data race we need to fix!
```

You also need to manually copy the `event.Rune() == 'f'` if-block from `../inputhandlers.go` into `main.go` if it changed
