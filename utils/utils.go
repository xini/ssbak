package utils

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/axllent/ssbak/app"
)

// IsFile returns if a path is a file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// IsDir returns if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}

// MkDirIfNotExists will create a directory if it doesn't exist
func MkDirIfNotExists(path string) error {
	if !IsDir(path) {
		app.Log(fmt.Sprintf("Creating temporary directory '%s'", path))
		return os.MkdirAll(path, os.ModePerm)
	}

	return nil
}

// CalcSize returns the size of a directory or file
func CalcSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// Convert an int64 to uint64
func int64Touint64(val int64) uint64 {
	return uint64(val)
}

// HasEnoughSpace will return an error message if the provided path does not
// have sufficient storage space
func HasEnoughSpace(path string, requiredSize int64) error {
	if runtime.GOOS == "windows" {
		// we don't check on Windows
		return nil
	}

	var stat syscall.Statfs_t

	syscall.Statfs(path, &stat)

	// Available blocks * size per block = available space in bytes
	remainingBytes := stat.Bavail * uint64(stat.Bsize)

	storageExpected := uint64(requiredSize)

	if storageExpected > remainingBytes {
		return fmt.Errorf("%s does not have enough space available (+-%s required)", path, ByteToHr(requiredSize))
	}

	return nil
}

// ByteToHr returns a human readable size as a string
func ByteToHr(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// GzipFile will compress an existing file with gzip and save it was output
func GzipFile(file, output string) error {
	src, err := os.Open(file)
	if err != nil {
		return err
	}
	defer src.Close()

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := bufio.NewWriter(outFile)
	defer buf.Flush()

	gz := gzip.NewWriter(buf)
	defer gz.Close()

	inSize, _ := CalcSize(file)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", file, ByteToHr(inSize), output))

	_, err = io.Copy(gz, src)

	outSize, _ := CalcSize(output)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", output, ByteToHr(outSize)))

	return err
}

// Which locates a binary in the current $PATH.
// It will append ".exe" to the filename if the platform is Windows.
func Which(binName string) (string, error) {
	if runtime.GOOS == "windows" {
		// append ".exe" to binary name if Windows
		binName += ".exe"
	}

	return exec.LookPath(binName)
}

// SkipResampled detects whether the assets is a resampled image
func skipResampled(filePath string) bool {
	if !app.IgnoreResampled {
		return false
	}

	for _, r := range app.ResampledRegex {
		if r.MatchString(filePath) {
			return true
		}
	}

	return false
}
