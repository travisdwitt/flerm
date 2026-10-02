package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"flerm/internal/canvas"
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

func findChart(name string) string {
	names := []string{name}
	if filepath.Ext(name) == "" {
		names = nil
		for _, ext := range canvas.ChartExts {
			names = append(names, name+ext)
		}
	}
	dirs := []string{""}
	if dir := config.Load().SaveDirectory; dir != "" {
		dirs = append(dirs, dir)
	}
	for _, n := range names {
		for _, dir := range dirs {
			path := filepath.Join(dir, n)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	return ""
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

func describeChart(name string) error {
	path := findChart(name)
	if path == "" {
		return fmt.Errorf("chart not found: %s", name)
	}
	c := canvas.NewCanvas()
	if _, _, err := c.LoadFromFileWithPan(path); err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	c.Describe(os.Stdout, path)
	return nil
}

func main() {
	dateOverride, rest := takeDateOverride(os.Args[1:])
	os.Args = append(os.Args[:1], rest...)

	noResume := flag.Bool("no-resume", false, "Don't offer the r:esume function in the main Flerm menu.")
	describe := flag.String("describe", "", "Print a chart's details in plain text so screen readers can describe it (experimental).")
	setup := flag.Bool("setup", false, "Create ~/.flermrc config with all the defaults pre-set.")
	flag.Parse()

	if *setup {
		if err := writeDefaultConfig(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *describe != "" {
		if err := describeChart(*describe); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := tui.Run(*noResume, dateOverride); err != nil {
		log.Fatal(err)
	}
}
