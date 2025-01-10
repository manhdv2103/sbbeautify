package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/muesli/termenv"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	output := termenv.NewOutput(os.Stdout)
	linePrinted := false
	t := time.Now()

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
			if beautifulLine, success := beautify(output, line); success {
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
