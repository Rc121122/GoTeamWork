package clip_helper

import (
	"archive/tar"
	"bufio"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver/v3"
)

// ProgressCallback is called during archiving to report progress
type ProgressCallback func(processedBytes int64, totalBytes int64, currentFile string)

// TarPaths archives the given file and directory paths into a tarball written to writer without compression.
// Optimized for large files with direct streaming and larger buffers.
func TarPaths(paths []string, writer io.Writer) error {
	return TarPathsWithProgress(paths, writer, nil)
}

// TarPathsWithProgress archives with optional progress reporting
func TarPathsWithProgress(paths []string, writer io.Writer, progress ProgressCallback) error {
	// Use buffered writer for better performance with large files
	bufWriter := bufio.NewWriterSize(writer, 64*1024*1024) // 64MB buffer
	tw := tar.NewWriter(bufWriter)
	defer func() {
		tw.Close()
		bufWriter.Flush()
	}()

	// Process paths sequentially to keep tar stream consistent
	for _, p := range paths {
		if err := addPathToTarWithProgress(tw, p, progress); err != nil {
			return err
		}
	}

	return nil
}

func addPathToTar(tw *tar.Writer, srcPath string) error {
	return addPathToTarWithProgress(tw, srcPath, nil)
}

func addPathToTarWithProgress(tw *tar.Writer, srcPath string, progress ProgressCallback) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	baseDir := filepath.Dir(srcPath)
	// When walking a single directory, ensure the directory name is preserved
	if info.IsDir() && len(srcPath) > 0 {
		baseDir = filepath.Dir(srcPath)
	}

	var totalProcessed int64
	var lastProgress time.Time

	return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		hdr.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Use buffered reader for better performance with large files
		bufReader := bufio.NewReaderSize(file, 64*1024*1024) // 64MB buffer

		// Create a progress writer if progress callback is provided
		var writer io.Writer = tw
		if progress != nil {
			writer = &progressWriter{
				writer:   tw,
				progress: progress,
				fileName: filepath.Base(path),
				total:    &totalProcessed,
				lastTime: &lastProgress,
			}
		}

		_, err = io.Copy(writer, bufReader)
		return err
	})
}

// TarPathsFast uses a high-performance third-party library for potentially faster archiving
func TarPathsFast(paths []string, writer io.Writer) error {
	// Create a temporary file for the archiver
	tmpFile, err := os.CreateTemp("", "fast_tar_*.tar")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create a tar archiver without compression for maximum speed
	tarArchiver := archiver.NewTar()

	// Archive to temp file
	if err := tarArchiver.Archive(paths, tmpFile.Name()); err != nil {
		return err
	}

	// Copy the result to the writer
	tmpFile.Seek(0, 0)
	_, err = io.Copy(writer, tmpFile)
	return err
}

// progressWriter wraps a writer to report progress
type progressWriter struct {
	writer   io.Writer
	progress ProgressCallback
	fileName string
	total    *int64
	lastTime *time.Time
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	if n > 0 {
		*pw.total += int64(n)
		// Throttle progress updates to avoid overwhelming UI
		now := time.Now()
		if pw.lastTime.IsZero() || now.Sub(*pw.lastTime) > 100*time.Millisecond {
			*pw.lastTime = now
			if pw.progress != nil {
				pw.progress(*pw.total, 0, pw.fileName) // 0 for total since we don't know total ahead of time
			}
		}
	}
	return n, err
}
