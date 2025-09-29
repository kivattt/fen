//go:build gen
// +build gen

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

const Package = "github.com/charlievieth/strcase"

var modRe = regexp.MustCompile(`(?m)^module[ ]+` + regexp.QuoteMeta(Package) + `$`)

func isStrcaseModule(name string) (bool, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return false, err
	}
	return modRe.Match(data), nil
}

func findModfile(child string) (string, error) {
	if !filepath.IsAbs(child) {
		return child, errors.New("directory must be absolute: " + child)
	}
	var first error
	dir := filepath.Clean(child)
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			path := filepath.Join(dir, "go.mod")
			ok, err := isStrcaseModule(path)
			if err != nil {
				if first == nil {
					first = err
				}
				continue
			}
			if ok {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if len(parent) >= len(dir) {
			break
		}
		dir = parent
	}
	if first != nil {
		return child, fmt.Errorf("util: error finding go.mod for package %q "+
			"in directory: %q: %w", Package, child, first)
	}
	return child, fmt.Errorf("util: failed to find go.mod for package %q "+
		"in directory: %q", Package, child)
}

var projectRoot = func() func() string {
	var root string
	var once sync.Once
	return func() string {
		once.Do(func() {
			wd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			dir, err := findModfile(wd)
			if err != nil {
				log.Fatal(err)
			}
			root = dir
		})
		return root
	}
}()

func buildGen() string {
	root := projectRoot()

	// Create "bin" directory
	if err := os.Mkdir(filepath.Join(root, "bin"), 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	gendir := filepath.Join(root, "internal/gen/gentables")
	if _, err := os.Stat(gendir); err != nil {
		log.Fatal(err)
	}

	exe := filepath.Join(root, "bin", "gentables")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	// Try make first since it's better at avoiding unnecessary builds.
	if mk, err := exec.LookPath("make"); err == nil {
		cmd := exec.Command(mk, "bin/gentables")
		cmd.Dir = root
		if cmd.Run() == nil {
			return exe
		}
	}

	cmd := exec.Command("go", "build", "-o", exe)
	cmd.Dir = gendir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running command %q: %v", cmd.Args, err)
	}
	return exe
}

func genCmd(args ...string) int {
	root := projectRoot()
	cmd := exec.Command(buildGen(), args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("error running command %q: %v", cmd.Args, err)
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		return 3
	}
	return 0
}

func usage() {
	const msg = "Usage: %[1]s [GENTABLES OPTION...]\n" +
		"\n" +
		"%[1]s is a wrapper for running the generation tool internal/gen/gentables.\n" +
		"It builds internal/gen/gentables then runs it for each supported Unicode version\n" +
		"with the provided args (which are passed to gentables directly)."
	fmt.Fprintf(os.Stderr, msg, filepath.Base(os.Args[0]))
	os.Exit(1)
}

func realMain(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "-help", "--help":
			usage()
			return 0
		}
	}
	var exitcode int
	// Supporting Unicode version 12.0.0 is annoying since arm64 support
	// is lacking on Go 1.15 and below.
	for _, version := range []string{"13.0.0", "15.0.0"} {
		code := genCmd(append([]string{"-unicode", version}, args...)...)
		if exitcode == 0 {
			exitcode = code
		}
	}
	return exitcode
}

func main() {
	log.SetPrefix("gen: ")
	log.SetFlags(log.Lshortfile)
	if code := realMain(os.Args[1:]); code != 0 {
		log.Fatal("exit:", code)
	}
}
