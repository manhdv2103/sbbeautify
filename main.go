package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/muesli/termenv"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	output := termenv.NewOutput(os.Stdout)

	for scanner.Scan() {
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
