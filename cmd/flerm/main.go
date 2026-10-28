package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"flerm/internal/agent"
	"flerm/internal/config"
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

func writeDefaultConfig() error {
	path, err := config.Path()
	if err != nil {
		return fmt.Errorf("what in the world is in that [home_directory]? What you got in that [home_directory]: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists. This is a first-time setup so you need to delete your .flermrc first so we can start fresh.", path)
	}
	if err := config.Defaults().Save(path); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("Wrote %s with the default configuration. Have fun ( ͡° ͜ʖ ͡°)\n", path)
	return nil
}

func runOp(name string, args []string) {
	if name == "mcp" {
		if err := agent.Serve(os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}
	p, err := agent.ParseArgs(name, args)
	if err != nil {
		log.Fatal(err)
	}
	out, err := agent.Ops[name].Run(p)
	fmt.Print(out)
	if out != "" && out[len(out)-1] != '\n' {
		fmt.Println()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) > 1 {
		if _, ok := agent.Ops[os.Args[1]]; ok || os.Args[1] == "mcp" {
			runOp(os.Args[1], os.Args[2:])
			return
		}
	}

	dateOverride, rest := takeDateOverride(os.Args[1:])
	os.Args = append(os.Args[:1], rest...)

	noResume := flag.Bool("no-resume", false, "Don't offer the r:esume function in the main Flerm menu.")
	describe := flag.String("describe", "", "Print a chart's details in plain text so screen readers can describe it.")
	format := flag.String("format", "text", "With --describe: text or json.")
	setup := flag.Bool("setup", false, "Create ~/.flermrc config with all the defaults pre-set.")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), "flerm - a flowchart editor for the terminal\n\nUsage:\n  flerm [flags] [file]    open the editor, optionally on a chart (a path, or a name from the current/save directory)\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *setup {
		if err := writeDefaultConfig(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *describe != "" {
		runOp("describe", []string{*describe, "--format", *format})
		return
	}

	var file string
	if name := flag.Arg(0); name != "" {
		if file = agent.FindChart(name); file == "" {
			log.Fatalf("chart not found: %s", name)
		}
		file, _ = filepath.Abs(file)
	}

	if err := tui.Run(*noResume, dateOverride, file); err != nil {
		log.Fatal(err)
	}
}
