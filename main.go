package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	output := termenv.NewOutput(os.Stdout)
	linePrinted := false
	t := time.Now()

	basePackage, err := getProjectBasePackage()

	warningLabel := output.String("Warning:").Foreground(output.Color("3")).Bold()
	if err != nil {
		fmt.Println(warningLabel, "cannot determine project's base package:", err)
	} else if basePackage == "" {
		fmt.Println(warningLabel, "cannot determine project's base package")
	}

	for scanner.Scan() {
		now := time.Now()
		duration := now.Sub(t)
		if linePrinted && duration.Seconds() > 5 {
			fmt.Println()
			fmt.Println(output.String(
				fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫ After %s", duration),
			).Bold())
			fmt.Println()
		}
		linePrinted = true
		t = now

		line := scanner.Text()
		beautifySuccess := false
		for _, beautify := range BEAUTIFIERS {
			if beautifulLine, success := beautify(output, line, basePackage); success {
				beautifySuccess = true
				fmt.Printf("%s", beautifulLine)
				break
			}
		}

		if !beautifySuccess {
			fmt.Printf("%s", line)
		}

		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %v\n", err)
	}
}

func getProjectBasePackage() (string, error) {
	startDir := filepath.Join("src", "main", "java")
	basePackage := ""

	err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			if len(entries) > 1 {
				relPath, _ := filepath.Rel(startDir, path)
				basePackage = strings.ReplaceAll(relPath, "/", ".")
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipDir {
		return "", err
	}

	return basePackage, nil
}
