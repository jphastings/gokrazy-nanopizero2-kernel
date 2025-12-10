package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed config.txt
var configContents []byte

// NanoPi Zero2 DTS was merged in 6.18
var kernelURL = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.18.tar.xz"

func downloadKernel() error {
	filename := filepath.Base(kernelURL)
	if _, err := os.Stat(filename); err == nil {
		log.Printf("kernel archive already exists, skipping download")
		return nil
	}

	log.Printf("downloading %s", kernelURL)
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(kernelURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, kernelURL)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func compile() error {
	// Start with defconfig
	defconfig := exec.Command("make", "ARCH=arm64", "defconfig")
	defconfig.Stdout = os.Stdout
	defconfig.Stderr = os.Stderr
	if err := defconfig.Run(); err != nil {
		return fmt.Errorf("make defconfig: %v", err)
	}

	// Append our config overlay
	f, err := os.OpenFile(".config", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(configContents); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Resolve config
	olddefconfig := exec.Command("make", "ARCH=arm64", "olddefconfig")
	olddefconfig.Stdout = os.Stdout
	olddefconfig.Stderr = os.Stderr
	if err := olddefconfig.Run(); err != nil {
		return fmt.Errorf("make olddefconfig: %v", err)
	}

	// Build kernel and DTBs
	make := exec.Command("make", "Image", "dtbs", "-j"+strconv.Itoa(runtime.NumCPU()))
	make.Env = append(os.Environ(),
		"ARCH=arm64",
		"CROSS_COMPILE=aarch64-linux-gnu-",
		"KBUILD_BUILD_USER=gokrazy",
		"KBUILD_BUILD_HOST=docker",
	)
	make.Stdout = os.Stdout
	make.Stderr = os.Stderr
	if err := make.Run(); err != nil {
		return fmt.Errorf("make: %v", err)
	}

	return nil
}

func copyFile(dest, src string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	return err
}

func main() {
	if err := downloadKernel(); err != nil {
		log.Fatal(err)
	}

	log.Printf("unpacking kernel")
	untar := exec.Command("tar", "xf", filepath.Base(kernelURL))
	untar.Stdout = os.Stdout
	untar.Stderr = os.Stderr
	if err := untar.Run(); err != nil {
		log.Fatal(err)
	}

	srcdir := strings.TrimSuffix(filepath.Base(kernelURL), ".tar.xz")
	if err := os.Chdir(srcdir); err != nil {
		log.Fatal(err)
	}

	log.Printf("compiling kernel")
	if err := compile(); err != nil {
		log.Fatal(err)
	}

	// Copy outputs
	if err := copyFile("/tmp/buildresult/vmlinuz", "arch/arm64/boot/Image"); err != nil {
		log.Fatal(err)
	}

	dtbSrc := "arch/arm64/boot/dts/rockchip/rk3528-nanopi-zero2.dtb"
	if _, err := os.Stat(dtbSrc); err != nil {
		log.Fatalf("DTB not found: %s - kernel version may not include NanoPi Zero2 support yet", dtbSrc)
	}
	if err := copyFile("/tmp/buildresult/rk3528-nanopi-zero2.dtb", dtbSrc); err != nil {
		log.Fatal(err)
	}
}
