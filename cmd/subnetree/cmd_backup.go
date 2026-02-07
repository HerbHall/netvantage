package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/HerbHall/subnetree/internal/backup"
)

func runBackup(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	output := fs.String("output", "", "output file path (default: subnetree-backup-{timestamp}.tar.gz)")
	dataDir := fs.String("data-dir", ".", "directory containing the database")
	configFile := fs.String("config", "", "path to config file to include in backup")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	dbPath := filepath.Join(*dataDir, "subnetree.db")

	if *output == "" {
		*output = fmt.Sprintf("subnetree-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
	}

	ctx := context.Background()
	if err := backup.Backup(ctx, dbPath, *configFile, *output); err != nil {
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Backup created: %s\n", *output)
}
