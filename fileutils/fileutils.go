// package fileutils provides utililty methods to copy and move files and directories.
package fileutils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// https://golang.org/cmd/go/#hdr-File_types
var goFileTypes = []string{
	".go",
	".c", ".h",
	".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx",
	".m",
	".s", ".S",
	".swig", ".swigcxx",
	".syso",
}

// Copypath copies the contents of src to dst, excluding any file that is not
// relevant to the Go compiler.
func Copypath(dst string, src string, tests bool) error {
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// https://golang.org/cmd/go/#hdr-Description_of_package_lists
		name := filepath.Base(path)
		if strings.HasPrefix(name, ".") ||
			(strings.HasPrefix(name, "_") && name != "_testdata") ||
			(!tests && name == "_testdata") ||
			(!tests && name == "testdata") ||
			(!tests && strings.HasSuffix(name, "_test.go")) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relevantFile := false
		for _, ext := range goFileTypes {
			if strings.HasSuffix(name, ext) {
				relevantFile = true
				break
			}
		}
		if !relevantFile {
			return nil
		}

		dst := filepath.Join(dst, path[len(src):])

		if info.Mode()&os.ModeSymlink != 0 {
			return Copylink(dst, path)
		}

		return Copyfile(dst, path)
	})
	if err != nil {
		// if there was an error during copying, remove the partial copy.
		RemoveAll(dst)
	}
	return err
}

func Copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	return err
}

func Copylink(dst, src string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("copylink: readlink: %v", err)
	}
	if err := mkdir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("copylink: mkdirall: %v", err)
	}
	if err := os.Symlink(target, dst); err != nil {
		return fmt.Errorf("copylink: symlink: %v", err)
	}
	return nil
}

// RemoveAll removes path and any children it contains. Unlike os.RemoveAll it
// deletes read only files on Windows.
func RemoveAll(path string) error {
	if runtime.GOOS == "windows" {
		// Simple case: if Remove works, we're done.
		err := os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		// make sure all files are writable so we can delete them
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// walk gave us some error, give it back.
				return err
			}
			mode := info.Mode()
			if mode|0200 == mode {
				return nil
			}
			return os.Chmod(path, mode|0200)
		})
	}
	return os.RemoveAll(path)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}