package main

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// SortMode controls the order of output records.
type SortMode string

const (
	SortByTime    SortMode = "time"
	SortByCommand SortMode = "command"
)

// Options drives a single tidy run.
type Options struct {
	// InputPath is both the source and the destination — the file is
	// rewritten in place after a timestamped backup is created.
	InputPath string
	// BackupDir is where pre-modification snapshots are kept.
	BackupDir string
	// Now stamps the backup filename. Injected so tests can pin it.
	Now func() time.Time
	// Sort controls output order.
	Sort SortMode
	// KeepSecrets disables the credential-redaction filter. Off by default;
	// turn on only when you've audited what's about to land on disk.
	KeepSecrets bool
}

// Stats summarises what a tidy run did.
type Stats struct {
	Read       int    // entries parsed successfully from input
	Skipped    int    // malformed entries skipped at parse time
	Redacted   int    // entries dropped because they looked like secrets
	Kept       int    // entries written to output after dedup
	BackupPath string // where the pre-modification snapshot was written
}

// Run executes one pass: back up the input, then parse → dedup (keeping the
// last occurrence of each command) → sort → atomically rewrite the input.
func Run(opts Options) (Stats, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}

	// Parse first — if the input is unreadable or invalid, fail before
	// touching anything on disk.
	in, err := os.Open(opts.InputPath)
	if err != nil {
		return Stats{}, fmt.Errorf("open input: %w", err)
	}
	records, skipped, err := ParseAll(in)
	_ = in.Close()
	if err != nil {
		return Stats{}, fmt.Errorf("parse: %w", err)
	}

	var redacted int
	deduped := dedupKeepLast(records)
	if err := sortRecords(deduped, opts.Sort); err != nil {
		return Stats{}, err
	}

	backup, err := backupFile(opts.InputPath, opts.BackupDir, opts.Now())
	if err != nil {
		return Stats{}, fmt.Errorf("backup: %w", err)
	}

	if err := writeAtomic(opts.InputPath, deduped); err != nil {
		return Stats{}, fmt.Errorf("write output: %w", err)
	}

	return Stats{
		Read:       len(records) + redacted,
		Skipped:    skipped,
		Redacted:   redacted,
		Kept:       len(deduped),
		BackupPath: backup,
	}, nil
}

// dedupKeepLast returns records with duplicate commands collapsed to the
// last occurrence (preserving its timestamp and original position).
func dedupKeepLast(records []Record) []Record {
	lastIdx := make(map[string]int, len(records))
	for i, r := range records {
		lastIdx[r.Command] = i
	}
	out := make([]Record, 0, len(lastIdx))
	for i, r := range records {
		if lastIdx[r.Command] == i {
			out = append(out, r)
		}
	}
	return out
}

func sortRecords(records []Record, mode SortMode) error {
	switch mode {
	case SortByTime:
		slices.SortStableFunc(records, func(a, b Record) int {
			return cmp.Compare(a.Timestamp, b.Timestamp)
		})
	case SortByCommand:
		slices.SortStableFunc(records, func(a, b Record) int {
			return cmp.Compare(a.Command, b.Command)
		})
	default:
		return fmt.Errorf("invalid --sort value %q (want: time|command)", string(mode))
	}
	return nil
}

// backupFile copies src into backupDir as
// "<basename>.<YYYYMMDD-HHMMSS>.bak" and returns the full path. If a file
// with that name already exists (same-second invocation), a numeric suffix
// is appended so a backup is never silently overwritten.
func backupFile(src, backupDir string, now time.Time) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	base := filepath.Base(src)
	stamp := now.Format("20060102-150405")

	dst := filepath.Join(backupDir, fmt.Sprintf("%s.%s.bak", base, stamp))
	for i := 1; ; i++ {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", err
		}
		dst = filepath.Join(backupDir, fmt.Sprintf("%s.%s-%d.bak", base, stamp, i))
	}

	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return "", err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	return dst, nil
}

// writeAtomic writes the records to dst by writing to a sibling temp file
// and renaming. The temp file is removed if anything fails before rename.
func writeAtomic(dst string, records []Record) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".zsh-history-tidy-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if err := writeRecords(tmp, records); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		cleanup()
		return err
	}
	return nil
}

func writeRecords(w io.Writer, records []Record) error {
	for _, r := range records {
		if _, err := io.WriteString(w, r.Marshal()); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}
