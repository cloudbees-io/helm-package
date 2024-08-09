package main

import (
	"fmt"
	"os"
)

func main() {
	err := Execute(os.Stdout)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cbhelmpkg: %s\n", err)
		os.Exit(1)
	}
}
