package main

import (
    "os"
    "fmt"

    "github.com/blackhillsinfosec/skyhook/cmd"
)

// Main initializes Gin variables, configures server options, and starts the file server.
func main() {

    // Handle CLI arguments and config file options
    if err := cmd.RootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
