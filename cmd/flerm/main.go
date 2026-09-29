package main

import (
	"flag"
	"log"
	"os"

	"flerm/internal/tui"
)

func takeDateOverride(args []string) (string, []string) {
	for i, arg := range args {
		if len(arg) != 5 || arg[0] != '-' {
			continue
		}
		digits := arg[1:]
		allDigits := true
		for _, r := range digits {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return digits, append(append([]string{}, args[:i]...), args[i+1:]...)
		}
	}
	return "", args
}

func main() {
	dateOverride, rest := takeDateOverride(os.Args[1:])
	os.Args = append(os.Args[:1], rest...)

	noResume := flag.Bool("no-resume", false, "don't offer to resume the last chart on the start menu")
	flag.Parse()
	if err := tui.Run(*noResume, dateOverride); err != nil {
		log.Fatal(err)
	}
}
