package main

//lint:file-ignore ST1005 some user-visible messages are stored in error values and thus occasionally require capitalization

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kivattt/gogitstatus"
	"github.com/rivo/tview"
)

type GitStatusHandler struct {
	app             *tview.Application
	fen             *Fen
	channel         chan string
	wg              sync.WaitGroup
	workerWaitGroup sync.WaitGroup
	ctx             context.Context
	cancelFunc      context.CancelFunc

	gitIndexFileWatcher *fsnotify.Watcher

	repoPathCurrentlyWorkingOn string // Does not require a mutex due to workerWaitGroup

	trackedLocalGitRepos      map[string]ChangedFileState
	trackedLocalGitReposMutex sync.Mutex
}

type ChangedFileState struct {
	changedFiles map[string]gogitstatus.ChangedFile
	lastChecked  time.Time
}

// Returns an error if path is not inside a tracked local git repository
func (gsh *GitStatusHandler) TryFindTrackedParentGitRepository(path string) (string, error) {
	gsh.trackedLocalGitReposMutex.Lock()
	defer gsh.trackedLocalGitReposMutex.Unlock()

	for repoPath := range gsh.trackedLocalGitRepos {
		// TODO: Improve performance? filepath.Rel() seems a little slow
		relativePathToRepo, err := filepath.Rel(repoPath, path)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(relativePathToRepo, "..") { // Hacky
			return repoPath, nil
		}
	}

	return "", errors.New("Path is not in any tracked git repositories")
}

// Looks for the first parent directory of path (or path itself) containing a ".git" directory.
// Returns an error if none found.
func (gsh *GitStatusHandler) TryFindParentGitRepository(path string) (string, error) {
	repoPathFound := path
	for {
		// Reached root path, no git repository found
		if repoPathFound == filepath.Dir(repoPathFound) {
			return "", errors.New("path is not in a local Git repository")
		}

		stat, err := os.Lstat(filepath.Join(repoPathFound, ".git"))
		if err == nil && stat.IsDir() {
			break
		}

		repoPathFound = filepath.Dir(repoPathFound)
	}

	return repoPathFound, nil
}

// Returns true if path is an unstaged/untracked file in the local Git repository at repositoryPath.
// Takes in absolute paths (panics when either are non-absolute).
func (gsh *GitStatusHandler) PathIsUnstagedOrUntracked(path, repositoryPath string) bool {
	if !filepath.IsAbs(path) || !filepath.IsAbs(repositoryPath) {
		panic("AbsolutePathIsUnstagedOrUntracked received a non-absolute path")
	}

	gsh.trackedLocalGitReposMutex.Lock()
	defer gsh.trackedLocalGitReposMutex.Unlock()

	repo, repoOk := gsh.trackedLocalGitRepos[repositoryPath]
	if !repoOk {
		return false
	}

	// TODO: Improve performance? filepath.Rel() seems a little slow
	relativePathToRepo, err := filepath.Rel(repositoryPath, path)
	if err != nil {
		return false
	}

	_, pathUnstagedOrUntracked := repo.changedFiles[relativePathToRepo]
	return pathUnstagedOrUntracked
}

func (gsh *GitStatusHandler) Init() {
	if gsh.app == nil {
		panic("In GitStatusHandler Init(), app was nil")
	}

	gsh.gitIndexFileWatcher, _ = fsnotify.NewWatcher()

	// Git index file watcher
	// This is so we update in real-time on "git add" / "git restore"
	go func() {
		for {
			select {
			case event, ok := <-gsh.gitIndexFileWatcher.Events:
				if !ok {
					return
				}

				// Git writes the new index to a temporary file called "index.lock"
				// which is then renamed to "index", resulting in a Create event for "index".
				// We need to ignore earlier events, so that we git status on the up-to-date index
				if !event.Op.Has(fsnotify.Create) || filepath.Base(event.Name) != "index" {
					continue
				}

				watchList := gsh.gitIndexFileWatcher.WatchList()
				if watchList != nil {
					if len(watchList) > 1 {
						panic("In GitStatusHandler: Length of watchList exceeded 1")
					}

					if len(watchList) == 1 {
						// TODO: Make this forcefully re-run the StatusWithContext()
						gsh.channel <- filepath.Dir(watchList[0])
					}
				}
			case _, ok := <-gsh.gitIndexFileWatcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	gsh.wg.Add(1)

	// Buffer size of 100 (arbitrary) prevents blocking when scrolling fast
	gsh.channel = make(chan string, 100)

	gsh.trackedLocalGitReposMutex.Lock()
	gsh.trackedLocalGitRepos = make(map[string]ChangedFileState)
	gsh.trackedLocalGitReposMutex.Unlock()

	go func() {
	chanLoop:
		for path := range gsh.channel {
			if !filepath.IsAbs(path) {
				panic("GitStatusHandler received a non-absolute path: \"" + path + "\"")
			}

			repoPathFound, err := gsh.TryFindParentGitRepository(path)
			if err != nil {
				continue chanLoop
			}

			// We don't want to restart a gogitstatus.StatusWithContext() already running on the same path
			if repoPathFound == gsh.repoPathCurrentlyWorkingOn {
				continue chanLoop
			}

			// Remove oldest tracked repository after 15 repos (pretty arbitrary, repos can vary massively in size)
			gsh.trackedLocalGitReposMutex.Lock()
			if len(gsh.trackedLocalGitRepos) > 15 {
				var oldestRepositoryTime time.Time
				oldestRepositoryPath := ""

				for k, v := range gsh.trackedLocalGitRepos {
					if oldestRepositoryPath == "" {
						oldestRepositoryTime = v.lastChecked
						oldestRepositoryPath = k
						continue
					}

					if v.lastChecked.Before(oldestRepositoryTime) {
						oldestRepositoryTime = v.lastChecked
						oldestRepositoryPath = k
					}
				}

				delete(gsh.trackedLocalGitRepos, oldestRepositoryPath)
			}
			gsh.trackedLocalGitReposMutex.Unlock()

			// Cancel the previous gogitstatus.StatusWithContext()
			if gsh.cancelFunc != nil {
				gsh.cancelFunc()
				gsh.workerWaitGroup.Wait()
			}

			gsh.repoPathCurrentlyWorkingOn = repoPathFound

			gsh.ctx, gsh.cancelFunc = context.WithCancel(context.Background())

			go func() {
				defer func() {
					// Allow a git status restart on this path
					gsh.repoPathCurrentlyWorkingOn = ""
					gsh.cancelFunc()
					gsh.workerWaitGroup.Done()
				}()

				gsh.workerWaitGroup.Add(1)

				if !filepath.IsAbs(gsh.repoPathCurrentlyWorkingOn) {
					panic("GitStatusHandler tried to run StatusWithContext on a non-absolute path: \"" + gsh.repoPathCurrentlyWorkingOn + "\"")
				}

				gsh.fen.runningGitStatus = true
				gsh.app.QueueUpdateDraw(func() {})

				changedFiles, err := gogitstatus.StatusWithContext(gsh.ctx, gsh.repoPathCurrentlyWorkingOn)
				changedFiles = gogitstatus.IncludingDirectories(changedFiles)

				if err != nil {
					gsh.fen.runningGitStatus = false // Can't defer this because it has to run before QueueUpdateDraw()
					return
				}

				gsh.trackedLocalGitReposMutex.Lock()
				gsh.trackedLocalGitRepos[gsh.repoPathCurrentlyWorkingOn] = ChangedFileState{
					changedFiles: changedFiles,
					lastChecked:  time.Now(),
				}
				gsh.trackedLocalGitReposMutex.Unlock()

				gsh.fen.runningGitStatus = false // Can't defer this because it has to run before QueueUpdateDraw()
				gsh.app.QueueUpdateDraw(func() {})
			}()
		}
		gsh.wg.Done()
	}()
}
