package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "linkko-api",
	Short: "Linkko API - Multi-tenant transactional API",
	Long:  `A production-ready Go API with multi-issuer JWT auth, rate limiting, idempotency, and observability.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
