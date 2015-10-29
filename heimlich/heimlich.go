package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func swallow(self string, binary string, paths []string) error {
	sf, err := os.Open(self)
	if err != nil {
		return fmt.Errorf("open %s: %s", self, err)
	}
	hzp := binary + ".hzp"
	f, err := os.OpenFile(hzp, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("open %s: %s", hzp, err)
	}
	n, err := io.Copy(f, sf)
	if err != nil {
		sf.Close()
		return fmt.Errorf("copy %s->%s: %s", self, hzp, err)
	}
	if err := sf.Close(); err != nil {
		return fmt.Errorf("close %s: %s", self, err)
	}

	zw := zip.NewWriter(f)
	zw.SetOffset(n)
	if err := swallowFile(zw, binary); err != nil {
		f.Close()
		return fmt.Errorf("swallow %s: %s", binary, err)
	}
	for _, path := range paths {
		if err := swallowFile(zw, path); err != nil {
			f.Close()
			return fmt.Errorf("swallow %s: %s", path, err)
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return f.Close()
}

func swallowFile(zw *zip.Writer, path string) error {
	fmt.Printf("  archiving %s ...\n", path)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %s", path, err)
	}
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %s", path, err)
	}
	fh, err := zip.FileInfoHeader(fi)
	if err != nil {
		return fmt.Errorf("fileinfoheader %s: %s", path, err)
	}
	fh.Name = path
	w, err := zw.CreateHeader(fh)
	if err != nil {
		return fmt.Errorf("create header %s: %s", path, err)
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("archive %s: %s", path, err)
	}
	return nil
}

func extract(hzp string) (string, error) {
	root := filepath.Dir(hzp)

	f, err := os.Open(hzp)
	if err != nil {
		return "", err
	}
	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		return "", err
	}
	for _, file := range zr.File {
		extractFile(root, file)
	}
	return filepath.Abs(filepath.Join(root, zr.File[0].Name))
}

func extractFile(root string, file *zip.File) error {
	fmt.Printf("  extracting %s ...\n", file.Name)
	path := filepath.Join(root, file.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	perms := file.FileInfo().Mode().Perm()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perms)
	if err != nil {
		r.Close()
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		r.Close()
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return r.Close()
}

func extractAndRun(hzp string, args []string, env []string) error {
	exePath, err := extract(hzp)
	if err != nil {
		return err
	}
	fmt.Printf("executing: %s %s\n", exePath, strings.Join(args, " "))
	return syscall.Exec(exePath, args, env)
}

func usage() {
	fmt.Printf("heimlich BINARY [PATH...]\n")
	os.Exit(2)
}

func main() {
	if strings.HasSuffix(os.Args[0], ".hzp") {
		if err := extractAndRun(os.Args[0], os.Args, os.Environ()); err != nil {
			fmt.Printf("extract error: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) < 2 {
		usage()
	}

	self, err := exec.LookPath(os.Args[0])
	if err != nil {
		fmt.Printf("where is %s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
	if err := swallow(self, os.Args[1], os.Args[2:]); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
