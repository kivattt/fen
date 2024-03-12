package main

import (
	"os"
	"os/exec"
	"log"
)

func main() {
	cmd := exec.Command("vim")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
}
