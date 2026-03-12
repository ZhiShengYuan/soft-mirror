package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"file-host/internal/model"
)

// Store manages binary files on the filesystem.
type Store struct {
	rootAbs string
}

// New creates a new Store rooted at rootDir (resolved to absolute path).
// Creates the directory if it doesn't exist.
func New(rootDir string) (*Store, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolving data dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	return &Store{rootAbs: abs}, nil
}

// safePath joins parts under rootAbs and verifies no path traversal occurs.
func (s *Store) safePath(parts ...string) (string, error) {
	joined := filepath.Join(append([]string{s.rootAbs}, parts...)...)
	resolved, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	if !strings.HasPrefix(resolved, s.rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected")
	}
	return resolved, nil
}

// filename returns the binary filename for a program on a given OS.
func filename(program, osName string) string {
	if osName == "windows" {
		return program + ".exe"
	}
	return program
}

// PutBinary atomically writes binary content for the given program/version/os/arch.
func (s *Store) PutBinary(program, version, osName, arch string, r io.Reader, maxSize int64) error {
	dir, err := s.safePath(program, version, osName, arch)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	fname := filename(program, osName)
	finalPath, err := s.safePath(program, version, osName, arch, fname)
	if err != nil {
		return err
	}

	tmpPath := finalPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	limited := io.LimitReader(r, maxSize+1)
	n, err := io.Copy(f, limited)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("writing file: %w", err)
	}
	if n > maxSize {
		os.Remove(tmpPath)
		return fmt.Errorf("file exceeds maximum size of %d bytes", maxSize)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalizing file: %w", err)
	}
	return nil
}

// GetBinaryPath returns the absolute path to a stored binary, or error if not found.
func (s *Store) GetBinaryPath(program, version, osName, arch string) (string, error) {
	fname := filename(program, osName)
	path, err := s.safePath(program, version, osName, arch, fname)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("binary not found: %s %s %s/%s", program, version, osName, arch)
		}
		return "", fmt.Errorf("stating binary: %w", err)
	}
	return path, nil
}

// DeleteBinary removes a single binary file and its parent directories if empty.
func (s *Store) DeleteBinary(program, version, osName, arch string) error {
	fname := filename(program, osName)
	path, err := s.safePath(program, version, osName, arch, fname)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting binary: %w", err)
	}
	// Clean up empty parent dirs (arch, os, version) but not program dir
	archDir, _ := s.safePath(program, version, osName, arch)
	osDir, _ := s.safePath(program, version, osName)
	verDir, _ := s.safePath(program, version)
	for _, dir := range []string{archDir, osDir, verDir} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
	}
	return nil
}

// DeleteVersion removes an entire version directory.
func (s *Store) DeleteVersion(program, version string) error {
	dir, err := s.safePath(program, version)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting version: %w", err)
	}
	return nil
}

// ListVersions returns the version strings for a program (directory names).
// Returns empty slice if program doesn't exist.
func (s *Store) ListVersions(program string) ([]string, error) {
	dir, err := s.safePath(program)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading versions: %w", err)
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

// ListPrograms returns the names of all hosted programs.
func (s *Store) ListPrograms() ([]string, error) {
	entries, err := os.ReadDir(s.rootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading programs: %w", err)
	}
	var programs []string
	for _, e := range entries {
		if e.IsDir() {
			programs = append(programs, e.Name())
		}
	}
	return programs, nil
}

// ListPlatforms returns all available OS/arch combinations for a program version.
func (s *Store) ListPlatforms(program, version string) ([]model.Platform, error) {
	verDir, err := s.safePath(program, version)
	if err != nil {
		return nil, err
	}
	osEntries, err := os.ReadDir(verDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading platforms: %w", err)
	}

	var platforms []model.Platform
	for _, osEntry := range osEntries {
		if !osEntry.IsDir() {
			continue
		}
		osName := osEntry.Name()
		archDir, err := s.safePath(program, version, osName)
		if err != nil {
			continue
		}
		archEntries, err := os.ReadDir(archDir)
		if err != nil {
			continue
		}
		for _, archEntry := range archEntries {
			if !archEntry.IsDir() {
				continue
			}
			archName := archEntry.Name()
			// Verify the binary file actually exists
			fname := filename(program, osName)
			binPath, err := s.safePath(program, version, osName, archName, fname)
			if err != nil {
				continue
			}
			if _, err := os.Stat(binPath); err == nil {
				platforms = append(platforms, model.Platform{OS: osName, Arch: archName})
			}
		}
	}
	return platforms, nil
}

// BinaryInfo returns metadata for a stored binary.
func (s *Store) BinaryInfo(program, version, osName, arch string) (*model.Binary, error) {
	fname := filename(program, osName)
	path, err := s.safePath(program, version, osName, arch, fname)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("binary not found")
		}
		return nil, fmt.Errorf("stat binary: %w", err)
	}
	return &model.Binary{
		Program: program,
		Version: version,
		OS:      osName,
		Arch:    arch,
		Size:    info.Size(),
		ModTime: info.ModTime().UTC().Truncate(time.Second),
	}, nil
}
