package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/HerbHall/subnetree/internal/backup"
)

func runRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	input := fs.String("input", "", "backup archive to restore (required)")
	dataDir := fs.String("data-dir", ".", "target directory for restored files")
	force := fs.Bool("force", false, "overwrite existing files")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	if err := backup.Restore(ctx, *input, *dataDir, *force); err != nil {
		fmt.Fprintf(os.Stderr, "restore failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Restore complete: files restored to %s\n", *dataDir)
}
