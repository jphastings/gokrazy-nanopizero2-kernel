package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

const (
	ubootRepo = "https://github.com/u-boot/u-boot"
	ubootRev  = "v2025.10"
	rkbinRepo = "https://github.com/rockchip-linux/rkbin"
	rkbinRev  = "master"
)

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

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func run(dir string, args ...string) error {
	log.Printf("Running: %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func compile(ubootDir, rkbinDir string) error {
	// Configure for generic RK3528
	if err := run(ubootDir, "make", "ARCH=arm64", "generic-rk3528_defconfig"); err != nil {
		return fmt.Errorf("make defconfig: %v", err)
	}

	// Append config options:
	// - CONFIG_BOOTMETH_SCRIPT: Enable boot.scr script boot method
	// - CONFIG_CMD_SETEXPR: Enable setexpr command for boot.scr cmdline.txt loading
	f, err := os.OpenFile(filepath.Join(ubootDir, ".config"), os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	configOverlay := `
CONFIG_BOOTMETH_SCRIPT=y
CONFIG_CMD_SETEXPR=y
CONFIG_CMD_SETEXPR_FMT=y
`
	if _, err := f.WriteString(configOverlay); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Resolve config dependencies
	if err := run(ubootDir, "make", "ARCH=arm64", "olddefconfig"); err != nil {
		return fmt.Errorf("make olddefconfig: %v", err)
	}

	// Build U-Boot
	cmd := exec.Command("make", "-j"+strconv.Itoa(runtime.NumCPU()))
	cmd.Dir = ubootDir
	cmd.Env = append(os.Environ(),
		"ARCH=arm64",
		"CROSS_COMPILE=aarch64-linux-gnu-",
		fmt.Sprintf("BL31=%s/bin/rk35/rk3528_bl31_v1.20.elf", rkbinDir),
		fmt.Sprintf("ROCKCHIP_TPL=%s/bin/rk35/rk3528_ddr_1056MHz_v1.11.bin", rkbinDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make: %v", err)
	}

	return nil
}

func generateBootScr(ubootDir, bootCmdPath string) error {
	cmd := exec.Command(
		filepath.Join(ubootDir, "tools/mkimage"),
		"-A", "arm64",
		"-T", "script",
		"-C", "none",
		"-d", bootCmdPath,
		"boot.scr",
	)
	cmd.Dir = ubootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	ubootDir, err := os.MkdirTemp("", "u-boot")
	if err != nil {
		log.Fatal(err)
	}

	rkbinDir, err := os.MkdirTemp("", "rkbin")
	if err != nil {
		log.Fatal(err)
	}

	// Clone rkbin (contains BL31 and TPL binaries)
	for _, cmd := range [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", rkbinRepo},
		{"git", "fetch", "--depth=1", "origin", rkbinRev},
		{"git", "checkout", "FETCH_HEAD"},
	} {
		if err := run(rkbinDir, cmd...); err != nil {
			log.Fatal(err)
		}
	}

	// Clone U-Boot
	for _, cmd := range [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", ubootRepo},
		{"git", "fetch", "--depth=1", "origin", ubootRev},
		{"git", "checkout", "FETCH_HEAD"},
	} {
		if err := run(ubootDir, cmd...); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("compiling U-Boot")
	if err := compile(ubootDir, rkbinDir); err != nil {
		log.Fatal(err)
	}

	bootCmdPath, err := filepath.Abs("boot.cmd")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("generating boot.scr")
	if err := generateBootScr(ubootDir, bootCmdPath); err != nil {
		log.Fatal(err)
	}

	// Copy outputs
	for _, file := range []struct{ src, dest string }{
		{filepath.Join(ubootDir, "u-boot-rockchip.bin"), "/tmp/buildresult/u-boot-rockchip.bin"},
		{filepath.Join(ubootDir, "boot.scr"), "/tmp/buildresult/boot.scr"},
	} {
		if err := copyFile(file.dest, file.src); err != nil {
			log.Fatal(err)
		}
	}
}
