# Phase 1: Development Environment & Initial Boot

**Goal:** Build U-Boot and kernel using Docker for reproducible builds, create the gokrazy package structure, and verify boot on hardware.

This phase produces the same artifacts that will be checked into the repository - we build it correctly from the start using Docker, following the pattern established by [gokrazy-rock64-kernel](https://github.com/anupcshan/gokrazy-rock64-kernel).

## Prerequisites

- NanoPi Zero2 board
- MicroSD card (8GB+ recommended)
- USB-to-TTL serial adapter (must support 1.5 Mbaud - see Section 1)
- Docker or Podman installed
- Go 1.21+ installed

## Package Structure Overview

By the end of Phase 1, the repository will contain:

```text
gokrazy-nanopizero2-kernel/
├── cmd/
│   ├── gokr-build-kernel/      # Kernel build logic (runs in container)
│   │   ├── build.go
│   │   └── config.txt          # Kernel config overlay
│   ├── gokr-build-uboot/       # U-Boot build logic (runs in container)
│   │   └── build.go
│   ├── gokr-rebuild-kernel/    # Docker wrapper for kernel builds
│   │   └── kernel.go
│   └── gokr-rebuild-uboot/     # Docker wrapper for U-Boot builds
│       └── uboot.go
├── boot.cmd                    # U-Boot boot script source
├── boot.scr                    # Compiled boot script (generated)
├── cmdline.txt                 # Kernel command line (read at boot)
├── config.txt                  # Boot config (may be empty for Rockchip)
├── rk3528-nanopi-zero2.dtb     # Device tree blob (generated)
├── u-boot-rockchip.bin         # U-Boot binary (generated)
├── vmlinuz                     # Kernel image (generated)
├── kernel.go                   # Empty Go package for import
└── go.mod
```

## Step-by-Step Checklist

### 1. Serial Adapter Setup

> **Why:** The NanoPi Zero2 has no video output. Serial UART at 1,500,000 baud is the only way to see boot messages.

- [x] **1.1** Obtain a compatible USB-to-TTL adapter

  Recommended chipsets that support 1.5 Mbaud:

  - **FT232RL** (FTDI) - most reliable
  - **CP2104** (Silicon Labs)
  - **CH340G** - works but quality varies

  Avoid: PL2303-based adapters (many have baud rate limitations)

- [x] **1.2** Wire the adapter to the NanoPi Zero2 debug header

  The NanoPi Zero2 has an **8-pin 2.54mm header**:

  ```text
  Pin Layout (USB-C & Ethernet facing north):

  [1] [2]     1 = GND          2 = 5V
  [3] [4]     3 = UART_TX      4 = 5V
  [5] [6]     5 = UART_RX      6 = GND
  [7] [8]     7 = 3.3V         8 = GND
  ```

  Connect only these three wires:

  | Adapter | NanoPi Zero2      |
  | ------- | ----------------- |
  | GND     | Pin 1 (or 6 or 8) |
  | RX      | Pin 3 (TX)        |
  | TX      | Pin 5 (RX)        |

  > **Important:** Do NOT connect 5V from the adapter. Power the board via USB-C.

- [x] **1.3** Install a serial terminal program

  ```bash
  # macOS
  brew install picocom

  # Linux
  sudo apt install picocom
  ```

- [x] **1.4** Test the serial connection

  ```bash
  picocom -b 1500000 /dev/tty.usbserial-*   # macOS
  picocom -b 1500000 /dev/ttyUSB0           # Linux
  ```

  > With nothing connected, you should see a blank terminal. Characters you type won't echo (that's normal). Exit with `Ctrl-A Ctrl-X`.

---

### 2. Install Docker

> **Why:** Docker provides reproducible builds across different host systems. The cross-compiler and all dependencies are containerized.

- [x] **2.1** Install Docker

  ```bash
  # macOS
  brew install --cask docker
  # Then launch Docker.app

  # Linux (Debian/Ubuntu)
  sudo apt install docker.io
  sudo usermod -aG docker $USER
  # Log out and back in
  ```

- [x] **2.2** Verify Docker works

  ```bash
  docker run --rm hello-world
  ```

---

### 3. Create Build Tools

> **Why:** We create Go programs that orchestrate Docker-based builds. This ensures anyone can rebuild the kernel with a single command.

- [x] **3.1** Create the U-Boot build tool

  Create `cmd/gokr-build-uboot/build.go`.

- [x] **3.2** Create the U-Boot rebuild wrapper

  Create `cmd/gokr-rebuild-uboot/uboot.go`

- [x] **3.3** Create the boot script

  Create `boot.cmd`

- [x] **3.4** Create cmdline.txt

  Create `cmdline.txt`

- [x] **3.5** Create empty `config.txt`

---

### 4. Build U-Boot

- [x] **4.1** Run the U-Boot build

  ```bash
  go run ./cmd/gokr-rebuild-uboot
  ```

  This will:
  - Build a Docker container with cross-compilation tools
  - Clone U-Boot and rkbin inside the container
  - Compile U-Boot with RK3528 support
  - Generate boot.scr from boot.cmd
  - Output `u-boot-rockchip.bin` and `boot.scr`

- [x] **4.2** Verify outputs

  ```bash
  ls -lh u-boot-rockchip.bin boot.scr
  # u-boot-rockchip.bin should be ~8 MB
  # boot.scr should be ~1 KB
  ```

#### Test: U-Boot Build Verification

```bash
file u-boot-rockchip.bin
# Expected: "data" (raw binary)

file boot.scr
# Expected: "u-boot legacy uImage, , ..."
```

---

### 5. Create Kernel Build Tools

- [ ] **5.1** Create kernel config overlay

  Create `cmd/gokr-build-kernel/config.txt`:

  ```text
  # Ensure /proc/config.gz is available
  CONFIG_IKCONFIG=y
  CONFIG_IKCONFIG_PROC=y

  # Disable unnecessary wireless modules
  CONFIG_BT=n
  CONFIG_CFG80211=n
  CONFIG_NFC=n
  CONFIG_WIRELESS=n

  # Performance
  CONFIG_CPU_FREQ_DEFAULT_GOV_PERFORMANCE=y
  CONFIG_DEBUG_KERNEL=n

  # Required for Tailscale
  CONFIG_TUN=y

  # Rockchip platform support
  CONFIG_ARCH_ROCKCHIP=y
  CONFIG_ROCKCHIP_PHY=y
  CONFIG_ROCKCHIP_THERMAL=y
  CONFIG_ROCKCHIP_IOMMU=y
  CONFIG_ROCKCHIP_IODOMAIN=y
  CONFIG_ROCKCHIP_PM_DOMAINS=y
  CONFIG_PWM_ROCKCHIP=y
  CONFIG_NVMEM_ROCKCHIP_EFUSE=y
  CONFIG_NVMEM_ROCKCHIP_OTP=y

  # MMC/eMMC support
  CONFIG_MMC_DW=y
  CONFIG_MMC_DW_ROCKCHIP=y
  CONFIG_MMC_SDHCI=y
  CONFIG_MMC_SDHCI_OF_DWCMSHC=y

  # Ethernet (GMAC)
  CONFIG_STMMAC_ETH=y
  CONFIG_STMMAC_PLATFORM=y
  CONFIG_DWMAC_GENERIC=y
  CONFIG_DWMAC_ROCKCHIP=y

  # Console
  CONFIG_CMDLINE="console=ttyS2,1500000"
  ```

- [ ] **5.2** Create the kernel build tool

  Create `cmd/gokr-build-kernel/build.go`:

  ```go
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

  // Update to appropriate kernel version with RK3528 NanoPi Zero2 DTS
  var kernelURL = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.tar.xz"

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
  ```

- [ ] **5.3** Create kernel rebuild wrapper

  Create `cmd/gokr-rebuild-kernel/kernel.go`:

  ```go
  package main

  import (
  	"flag"
  	"io"
  	"log"
  	"os"
  	"os/exec"
  	"os/user"
  	"path/filepath"
  	"text/template"
  )

  const dockerFileContents = `
  FROM debian:bookworm

  RUN apt-get update && apt-get install -y \
      crossbuild-essential-arm64 bc libssl-dev bison flex wget xz-utils

  COPY gokr-build-kernel /usr/bin/gokr-build-kernel

  RUN echo 'builduser:x:{{ .Uid }}:{{ .Gid }}:nobody:/:/bin/sh' >> /etc/passwd && \
      chown -R {{ .Uid }}:{{ .Gid }} /usr/src

  USER builduser
  WORKDIR /usr/src
  ENTRYPOINT /usr/bin/gokr-build-kernel
  `

  var dockerFileTmpl = template.Must(template.New("dockerfile").Parse(dockerFileContents))

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

  func getContainerExecutable() string {
  	for _, exe := range []string{"podman", "docker"} {
  		if _, err := exec.LookPath(exe); err == nil {
  			return exe
  		}
  	}
  	return "docker"
  }

  func main() {
  	flag.Parse()
  	executable := getContainerExecutable()

  	tmp, err := os.MkdirTemp("/tmp", "gokr-rebuild-kernel")
  	if err != nil {
  		log.Fatal(err)
  	}
  	defer os.RemoveAll(tmp)

  	// Build the gokr-build-kernel binary for Linux
  	cmd := exec.Command("go", "build", "-o", filepath.Join(tmp, "gokr-build-kernel"),
  		"./cmd/gokr-build-kernel")
  	cmd.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0")
  	cmd.Stderr = os.Stderr
  	if err := cmd.Run(); err != nil {
  		log.Fatalf("go build: %v", err)
  	}

  	// Create Dockerfile
  	u, err := user.Current()
  	if err != nil {
  		log.Fatal(err)
  	}

  	dockerFile, err := os.Create(filepath.Join(tmp, "Dockerfile"))
  	if err != nil {
  		log.Fatal(err)
  	}
  	if err := dockerFileTmpl.Execute(dockerFile, struct{ Uid, Gid string }{u.Uid, u.Gid}); err != nil {
  		log.Fatal(err)
  	}
  	dockerFile.Close()

  	// Build container
  	log.Printf("building %s container for kernel compilation", executable)
  	build := exec.Command(executable, "build", "--rm=true", "--tag=gokr-rebuild-kernel", ".")
  	build.Dir = tmp
  	build.Stdout = os.Stdout
  	build.Stderr = os.Stderr
  	if err := build.Run(); err != nil {
  		log.Fatalf("%s build: %v", executable, err)
  	}

  	// Run container
  	log.Printf("compiling kernel (this takes 10-30 minutes)")
  	run := exec.Command(executable, "run", "--rm", "-v", tmp+":/tmp/buildresult:Z", "gokr-rebuild-kernel")
  	run.Stdout = os.Stdout
  	run.Stderr = os.Stderr
  	if err := run.Run(); err != nil {
  		log.Fatalf("%s run: %v", executable, err)
  	}

  	// Copy results back
  	for _, file := range []string{"vmlinuz", "rk3528-nanopi-zero2.dtb"} {
  		if err := copyFile(file, filepath.Join(tmp, file)); err != nil {
  			log.Fatal(err)
  		}
  		log.Printf("wrote %s", file)
  	}
  }
  ```

---

### 6. Build Kernel

- [ ] **6.1** Run the kernel build

  ```bash
  go run ./cmd/gokr-rebuild-kernel
  ```

  This takes 10-30 minutes. It will:
  - Download the kernel source
  - Apply the config overlay
  - Cross-compile for ARM64
  - Output `vmlinuz` and `rk3528-nanopi-zero2.dtb`

- [ ] **6.2** Verify outputs

  ```bash
  ls -lh vmlinuz rk3528-nanopi-zero2.dtb
  # vmlinuz should be ~15-25 MB
  # DTB should be ~50-100 KB

  file vmlinuz
  # Expected: "Linux kernel ARM64 boot executable Image, ..."
  ```

---

### 7. Prepare Test SD Card

> **Why:** We test the built artifacts by booting on real hardware.

- [ ] **7.1** Identify SD card device

  ```bash
  # macOS
  diskutil list

  # Linux
  lsblk
  ```

  **Be very careful** - wrong device = data loss!

- [ ] **7.2** Write U-Boot to SD card

  ```bash
  # Unmount first
  diskutil unmountDisk /dev/diskX   # macOS
  sudo umount /dev/sdX*             # Linux

  # Write U-Boot at sector 64
  sudo dd if=u-boot-rockchip.bin of=/dev/rdiskX seek=64 bs=512   # macOS
  sudo dd if=u-boot-rockchip.bin of=/dev/sdX seek=64 bs=512      # Linux
  sudo sync
  ```

- [ ] **7.3** Create boot partition

  ```bash
  # macOS (use Disk Utility or)
  diskutil partitionDisk /dev/diskX GPT FAT32 BOOT 256MB "Free Space" 0

  # Linux
  sudo parted /dev/sdX --script mklabel gpt
  sudo parted /dev/sdX --script mkpart primary fat32 16MB 256MB
  sudo mkfs.vfat -F 32 /dev/sdX1
  ```

- [ ] **7.4** Copy boot files

  ```bash
  # Mount boot partition
  # macOS: mounts automatically as /Volumes/BOOT
  # Linux: sudo mount /dev/sdX1 /mnt

  cp vmlinuz /Volumes/BOOT/           # or /mnt/
  cp rk3528-nanopi-zero2.dtb /Volumes/BOOT/
  cp boot.scr /Volumes/BOOT/
  cp cmdline.txt /Volumes/BOOT/

  # Unmount
  diskutil unmount /Volumes/BOOT      # macOS
  sudo umount /mnt                     # Linux
  ```

---

### 8. Boot Test

- [ ] **8.1** Connect serial adapter and start terminal

  ```bash
  picocom -b 1500000 /dev/tty.usbserial-*
  ```

- [ ] **8.2** Insert SD card and power on

- [ ] **8.3** Watch for boot output

#### Test: Expected Serial Output

```text
DDR V1.10
LPDDR4X, 1056MHz
...
U-Boot TPL 2025.01-...
U-Boot SPL 2025.01-...

U-Boot 2025.01-...
Model: Generic RK3528
DRAM: 2 GiB
...
Loading kernel ...
Boot args: console=ttyS2,1500000 earlycon root=/dev/mmcblk0p2 rootwait panic=10 oops=panic init=/gokrazy/init
Booting kernel ...

[    0.000000] Booting Linux on physical CPU 0x0000000000 [0x410fd034]
[    0.000000] Linux version 6.x.0 ...
[    0.000000] Machine model: FriendlyElec NanoPi Zero2
...
[    1.xxx] ---[ end Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0) ]---
```

**The kernel panic is expected!** It confirms:

- ✅ U-Boot built correctly and boots
- ✅ boot.scr loads cmdline.txt correctly
- ✅ Kernel built correctly and boots
- ✅ Device tree is recognized
- ❌ No gokrazy root filesystem (that's Phase 2+)

---

## Troubleshooting

### No output at all

1. Check TX/RX wiring (try swapping)
2. Verify baud rate is exactly 1500000
3. Ensure adapter supports 1.5 Mbaud

### Docker build fails

- Ensure Docker daemon is running
- Check you have ~5GB free disk space
- Try `docker system prune` to free space

### DTB missing from kernel build

The NanoPi Zero2 DTS targets kernel v6.18+. Options:
1. Use linux-next instead of stable kernel
2. Manually add DTS from [Jonas Karlman's patches](https://lists.infradead.org/pipermail/linux-arm-kernel/2025-July/1045454.html)

### U-Boot shows but kernel fails to load

1. Check boot.scr was generated correctly
2. In U-Boot, run `fatls mmc 0:1` to verify files exist
3. Check kernel isn't too large for memory addresses

---

## Phase 1 Completion Criteria

- [ ] Docker-based U-Boot build completes successfully
- [ ] Docker-based kernel build completes successfully
- [ ] All artifacts exist: `u-boot-rockchip.bin`, `boot.scr`, `vmlinuz`, `rk3528-nanopi-zero2.dtb`
- [ ] Serial terminal connects at 1500000 baud
- [ ] U-Boot boots and shows DDR/DRAM info
- [ ] boot.scr correctly loads cmdline.txt
- [ ] Kernel boots and shows "Machine model: FriendlyElec NanoPi Zero2"
- [ ] Kernel panics with "Unable to mount root fs" (expected)

---

## Resources

- [gokrazy-rock64-kernel](https://github.com/anupcshan/gokrazy-rock64-kernel) - Reference implementation
- [U-Boot Rockchip Documentation](https://docs.u-boot.org/en/latest/board/rockchip/rockchip.html)
- [RK3528 Mainline Status Tracker](https://github.com/ziyao233/rk3528-mainline)
- [NanoPi Zero2 Wiki](https://wiki.friendlyelec.com/wiki/index.php/NanoPi_Zero2)
