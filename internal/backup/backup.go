// Package backup provides tar.gz-based backup and restore for SubNetree data.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver
)

// Backup creates a tar.gz archive containing the SQLite database and an
// optional config file. It performs a WAL checkpoint before copying the
// database to ensure consistency.
func Backup(_ context.Context, dbPath, configPath, outputPath string) error {
	// Verify database exists.
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("database file not found: %w", err)
	}

	// Checkpoint WAL to flush pending writes.
	if err := checkpointWAL(dbPath); err != nil {
		return fmt.Errorf("WAL checkpoint failed: %w", err)
	}

	// Create the output archive.
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add the database file.
	if err := addFileToTar(tw, dbPath, filepath.Base(dbPath)); err != nil {
		return fmt.Errorf("adding database to archive: %w", err)
	}

	// Add the config file if specified and it exists.
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := addFileToTar(tw, configPath, filepath.Base(configPath)); err != nil {
				return fmt.Errorf("adding config to archive: %w", err)
			}
		}
		// If the config file doesn't exist, skip silently.
	}

	return nil
}

// checkpointWAL opens the database, runs a TRUNCATE checkpoint to flush the
// WAL, and closes the connection.
func checkpointWAL(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// addFileToTar adds a single file to the tar archive under the given name.
func addFileToTar(tw *tar.Writer, filePath, archiveName string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = archiveName

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}
