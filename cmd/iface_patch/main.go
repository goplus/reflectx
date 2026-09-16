package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	check := flag.Bool("check", false, "report whether GOROOT is already patched; do not write")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `usage: iface_patch [-check] <goroot>

Patch a Go source tree for wasm goplus.ifacefuncval.
Then: cd <goroot>/src && ./make.bash
      export GOROOT=<goroot> PATH=$GOROOT/bin:$PATH
      go clean -cache
      GOOS=wasip1 GOARCH=wasm go test -exec wasmtime -tags goplus.ifacefuncval -v .
See cmd/iface_patch/README.md.
Supported versions: %s
`, strings.Join(supportedVersions(), ", "))
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	root := flag.Arg(0)
	ver, err := goVersion(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	v, err := loadVersion(ver)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	n, err := applyAll(root, v, *check)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *check {
		if n == 0 {
			fmt.Println("already patched")
			return
		}
		fmt.Printf("needs patch (%d files)\n", n)
		os.Exit(1)
	}
	if n == 0 {
		fmt.Println("already patched")
		return
	}
	fmt.Printf("patched %d files\n", n)
}

func goVersion(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read VERSION: %w", err)
	}
	ver := strings.TrimSpace(strings.Split(string(b), "\n")[0])
	if ver == "" {
		return "", fmt.Errorf("empty VERSION in %s", root)
	}
	return ver, nil
}
