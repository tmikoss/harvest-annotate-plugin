package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch":
		cmdFetch(os.Args[2:])
	case "update-notes":
		cmdUpdateNotes(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: harvest-cli <command> [options]

Commands:
  fetch [--date D] [--from F --to T]  Fetch your time entries (default: today)
  update-notes <id> <notes>            Update notes on a time entry`)
}
