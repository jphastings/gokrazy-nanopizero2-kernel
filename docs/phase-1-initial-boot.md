# Phase 1: Development Environment & Initial Boot

**Goal:** Achieve serial console boot on NanoPi Zero2 with mainline Linux kernel.

## Prerequisites

- NanoPi Zero2 board
- MicroSD card (8GB+ recommended)
- USB-to-TTL serial adapter (must support 1.5 Mbaud - see notes below)
- Linux or macOS development machine
- Dupont jumper wires (female-to-female for the debug header)

## Step-by-Step Checklist

### 1. Serial Adapter Setup

> **Why:** The NanoPi Zero2 has no video output. The only way to see boot messages is via the debug UART at 1,500,000 baud (1.5 Mbps). Most cheap adapters max out at 115200 baud.

- [ ] **1.1** Obtain a compatible USB-to-TTL adapter

  Recommended chipsets that support 1.5 Mbaud:
  - **FT232RL** (FTDI) - most reliable
  - **CP2104** (Silicon Labs)
  - **CH340G** - works but quality varies

  Avoid: PL2303-based adapters (many have baud rate limitations)

- [ ] **1.2** Wire the adapter to the NanoPi Zero2 debug header

  The NanoPi Zero2 has an **8-pin 2.54mm header**:

  ```text
  Pin Layout (top view, USB-C facing down):

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

- [ ] **1.3** Install a serial terminal program

  **Linux (Debian/Ubuntu):**

  ```bash
  sudo apt install minicom
  # or
  sudo apt install picocom
  ```

  **macOS:**

  ```bash
  brew install minicom
  # or
  brew install picocom
  ```

- [ ] **1.4** Configure and test the serial connection

  Find your serial device:

  ```bash
  ls /dev/ttyUSB*   # Linux
  ls /dev/tty.usbserial*   # macOS
  ```

  Connect with picocom (simpler):

  ```bash
  picocom -b 1500000 /dev/ttyUSB0
  ```

  Or minicom:
  
  ```bash
  minicom -s
  # Set: Serial Device = /dev/ttyUSB0
  # Set: Bps/Par/Bits = 1500000 8N1
  # Set: Hardware Flow Control = No
  # Save setup as dfl, then exit
  minicom
  ```

  > **Test:** With nothing connected, you should see a blank terminal. Characters you type won't echo (that's normal).

---

### 2. Install Cross-Compilation Toolchain

> **Why:** You're building ARM64 binaries on an x86_64 (or ARM Mac) machine. The cross-compiler produces code for a different CPU architecture than your host.

- [ ] **2.1** Install the ARM64 cross-compiler

  **Debian/Ubuntu:**

  ```bash
  sudo apt update
  sudo apt install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu
  sudo apt install bison flex libssl-dev bc
  ```

  **macOS (via Homebrew):**

  ```bash
  brew install aarch64-elf-gcc
  brew install bison flex openssl
  # Note: macOS cross-compilation is trickier; a Linux VM is recommended
  ```

  **Fedora:**

  ```bash
  sudo dnf install gcc-aarch64-linux-gnu binutils-aarch64-linux-gnu
  sudo dnf install bison flex openssl-devel bc
  ```

- [ ] **2.2** Verify the toolchain works

  ```bash
  aarch64-linux-gnu-gcc --version
  ```

  Expected output: version info for the cross-compiler (e.g., `aarch64-linux-gnu-gcc (Ubuntu 11.4.0-1ubuntu1~22.04) 11.4.0`)

---

### 3. Build U-Boot Bootloader

> **Why:** Rockchip boards use U-Boot as the bootloader. It initializes hardware (DDR memory, storage) and loads the Linux kernel. The "idbloader" contains the DDR training code needed before U-Boot can run.

- [ ] **3.1** Create a workspace directory

  ```bash
  mkdir -p ~/nanopi-zero2-build
  cd ~/nanopi-zero2-build
  ```

- [ ] **3.2** Clone U-Boot and Rockchip binary blobs

  ```bash
  git clone --depth 1 https://source.denx.de/u-boot/u-boot.git
  git clone --depth 1 https://github.com/rockchip-linux/rkbin.git
  ```

- [ ] **3.3** Set environment variables for RK3528 firmware

  ```bash
  cd u-boot
  export BL31=../rkbin/bin/rk35/rk3528_bl31_v1.18.elf
  export ROCKCHIP_TPL=../rkbin/bin/rk35/rk3528_ddr_1056MHz_v1.10.bin
  ```

  > **What are these?**
  > - `BL31`: ARM Trusted Firmware (runs in secure mode, required for ARM64)
  > - `ROCKCHIP_TPL`: DDR memory initialization code (proprietary binary)

- [ ] **3.4** Configure U-Boot for RK3528

  ```bash
  make generic-rk3528_defconfig
  ```

  This creates `.config` with settings for generic RK3528 boards.

- [ ] **3.5** Build U-Boot

  ```bash
  make CROSS_COMPILE=aarch64-linux-gnu- -j$(nproc)
  ```

  This takes 1-2 minutes. Watch for errors.

- [ ] **3.6** Verify build output

  ```bash
  ls -la u-boot-rockchip.bin
  ```

  You should see a file around 1-2 MB. This contains:
  - idbloader (TPL + SPL)
  - U-Boot proper
  - ATF (ARM Trusted Firmware)

#### Test: U-Boot Build Verification

```bash
# Check the binary was created
file u-boot-rockchip.bin
# Expected: "data" or similar (it's a raw binary)

# Check size is reasonable (should be 1-2 MB)
ls -lh u-boot-rockchip.bin
```

---

### 4. Build Mainline Linux Kernel

> **Why:** The mainline kernel (from kernel.org) includes RK3528 support as of v6.16+. We need kernel version 6.18+ to get the NanoPi Zero2 device tree.

- [ ] **4.1** Clone the mainline kernel

  ```bash
  cd ~/nanopi-zero2-build

  # Option A: Latest stable (if 6.18+ is released)
  git clone --depth 1 --branch v6.12 https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git

  # Option B: linux-next (for latest RK3528 patches)
  git clone --depth 1 https://git.kernel.org/pub/scm/linux/kernel/git/next/linux-next.git linux
  ```

  > **Note:** At time of writing, NanoPi Zero2 DTS is targeting v6.18. Check [RK3528 mainline status](https://github.com/ziyao233/rk3528-mainline) for current state. You may need linux-next or to cherry-pick patches.

- [ ] **4.2** Configure the kernel

  ```bash
  cd linux
  make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- defconfig
  ```

  This creates a generic ARM64 config. We'll refine it in Phase 2.

- [ ] **4.3** Build the kernel and device trees

  ```bash
  make ARCH=arm64 CROSS_COMPILE=aarch64-linux-gnu- -j$(nproc)
  ```

  This takes 10-30 minutes depending on your machine.

- [ ] **4.4** Verify build outputs

  ```bash
  ls -la arch/arm64/boot/Image
  ls -la arch/arm64/boot/dts/rockchip/rk3528-nanopi-zero2.dtb
  ```

  > **If the DTB is missing:** The NanoPi Zero2 device tree may not be in your kernel version yet. See Troubleshooting section.

#### Test: Kernel Build Verification

```bash
# Check kernel image exists and is reasonable size (15-25 MB)
ls -lh arch/arm64/boot/Image

# Check device tree exists
ls arch/arm64/boot/dts/rockchip/rk3528*.dtb

# Verify it's an ARM64 kernel
file arch/arm64/boot/Image
# Expected: "Linux kernel ARM64 boot executable Image"
```

---

### 5. Prepare Bootable SD Card

> **Why:** We'll create a minimal SD card with U-Boot, kernel, and device tree. No root filesystem yet - we just want to see the kernel boot and panic (which proves everything works up to that point).

- [ ] **5.1** Insert SD card and identify the device

  ```bash
  # Before inserting
  ls /dev/sd*

  # After inserting
  ls /dev/sd*

  # The new device is your SD card (e.g., /dev/sdb)
  # BE VERY CAREFUL - wrong device = data loss!
  ```

  On macOS: `diskutil list` (look for the SD card, e.g., `/dev/disk4`)

- [ ] **5.2** Unmount any auto-mounted partitions

  ```bash
  # Linux
  sudo umount /dev/sdb*

  # macOS
  diskutil unmountDisk /dev/disk4
  ```

- [ ] **5.3** Write U-Boot to SD card

  ```bash
  cd ~/nanopi-zero2-build/u-boot

  # Linux (replace sdX with your device!)
  sudo dd if=u-boot-rockchip.bin of=/dev/sdX seek=64 bs=512 conv=notrunc
  sudo sync

  # macOS (use raw device for speed)
  sudo dd if=u-boot-rockchip.bin of=/dev/rdiskX seek=64 bs=512
  sudo sync
  ```

  > **Why seek=64?** Rockchip bootrom expects the bootloader at sector 64 (32 KB offset). Sectors 0-63 are reserved.

- [ ] **5.4** Create a FAT32 boot partition

  ```bash
  # Linux
  sudo parted /dev/sdX --script mklabel gpt
  sudo parted /dev/sdX --script mkpart primary fat32 16MB 256MB
  sudo mkfs.vfat -F 32 /dev/sdX1

  # Mount it
  sudo mkdir -p /mnt/boot
  sudo mount /dev/sdX1 /mnt/boot
  ```

  > **Why start at 16MB?** Leaves room for U-Boot in the reserved area.

- [ ] **5.5** Copy kernel and device tree to boot partition

  ```bash
  cd ~/nanopi-zero2-build/linux

  sudo cp arch/arm64/boot/Image /mnt/boot/
  sudo cp arch/arm64/boot/dts/rockchip/rk3528-nanopi-zero2.dtb /mnt/boot/
  ```

- [ ] **5.6** Create U-Boot boot script

  Create a file `boot.cmd`:

  ```bash
  cat << 'EOF' | sudo tee /mnt/boot/boot.cmd
  setenv bootargs console=ttyS2,1500000 earlycon=uart8250,mmio32,0xff9f0000
  load mmc 0:1 ${kernel_addr_r} Image
  load mmc 0:1 ${fdt_addr_r} rk3528-nanopi-zero2.dtb
  booti ${kernel_addr_r} - ${fdt_addr_r}
  EOF
  ```

  Compile it to a U-Boot script:

  ```bash
  sudo mkimage -C none -A arm64 -T script -d /mnt/boot/boot.cmd /mnt/boot/boot.scr
  ```

  > **Install mkimage if needed:** `sudo apt install u-boot-tools`

- [ ] **5.7** Unmount and eject

  ```bash
  sudo umount /mnt/boot
  sudo sync
  ```

---

### 6. First Boot Test

> **Why:** This is the moment of truth. We expect to see U-Boot messages, then kernel boot messages, then a kernel panic (because there's no root filesystem).

- [ ] **6.1** Connect serial adapter to your computer

- [ ] **6.2** Start serial terminal

  ```bash
  picocom -b 1500000 /dev/ttyUSB0
  ```

- [ ] **6.3** Insert SD card into NanoPi Zero2

- [ ] **6.4** Apply power via USB-C

- [ ] **6.5** Watch for boot output

#### Test: Expected Serial Output

**Success looks like this (abbreviated):**

```
DDR V1.10
LPDDR4X, 1056MHz
channel[0] BW=16 Col=10 Bk=8 CS0 Row=15 CS1 Row=15 CS=2 Die BW=16

U-Boot TPL 2025.01-... (date)
U-Boot SPL 2025.01-... (date)

U-Boot 2025.01-... (date)
Model: Generic RK3528
DRAM:  2 GiB
...

Starting kernel ...

[    0.000000] Booting Linux on physical CPU 0x0000000000 [0x410fd034]
[    0.000000] Linux version 6.x.0 (...)
[    0.000000] Machine model: FriendlyElec NanoPi Zero2
...
[    1.234567] ---[ end Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0) ]---
```

**The kernel panic is expected!** It means:

- ✅ DDR initialized correctly
- ✅ U-Boot loaded and ran
- ✅ Kernel loaded and started
- ✅ Device tree was parsed
- ❌ No root filesystem (expected - we didn't provide one)

---

## Troubleshooting

### No output at all

1. **Check wiring:** TX/RX may be swapped. Try swapping them.
2. **Check baud rate:** Must be exactly 1500000.
3. **Check voltage:** The debug UART is 3.3V. 5V adapters can damage the board.
4. **Try another adapter:** Some adapters can't do 1.5 Mbaud.

### Garbled output

- Wrong baud rate (most likely)
- Hardware flow control enabled (disable it)
- Bad USB cable or adapter

### U-Boot shows but kernel doesn't load

1. Check file names match exactly (`Image`, `rk3528-nanopi-zero2.dtb`)
2. Check boot.scr was created properly
3. In U-Boot prompt, try manually:

   ```text
   fatls mmc 0:1
   ```

   to see what files U-Boot can see.

### Device tree file missing from kernel build

The NanoPi Zero2 DTS is in mainline as of the v6.18 merge window. If building an older kernel:

1. Check [linux-next](https://git.kernel.org/pub/scm/linux/kernel/git/next/linux-next.git) for the latest
2. Or manually add the DTS file from [Jonas Karlman's patches](https://lists.infradead.org/pipermail/linux-arm-kernel/2025-July/1045454.html)

### macOS: Can't set 1500000 baud

macOS serial drivers often don't support non-standard baud rates. Solutions:

- Use a Linux VM (recommended)
- Try a Docker container with USB passthrough
- Use a Raspberry Pi as a serial bridge

---

## Phase 1 Completion Criteria

Before moving to Phase 2, verify:

- [ ] Serial terminal connected and working at 1500000 baud
- [ ] U-Boot successfully built for RK3528
- [ ] Mainline kernel successfully built with RK3528 support
- [ ] U-Boot boots and shows DDR info + U-Boot banner
- [ ] Kernel loads and shows boot messages
- [ ] Kernel panic shows "Unable to mount root fs" (expected)
- [ ] Serial console shows "Machine model: FriendlyElec NanoPi Zero2"

---

## Resources

- [U-Boot Rockchip Documentation](https://docs.u-boot.org/en/latest/board/rockchip/rockchip.html)
- [RK3528 Mainline Status Tracker](https://github.com/ziyao233/rk3528-mainline)
- [NanoPi Zero2 Wiki](https://wiki.friendlyelec.com/wiki/index.php/NanoPi_Zero2)
- [Cross-compiling for ARM64](https://jensd.be/1126/linux/cross-compiling-for-arm-or-aarch64-on-debian-or-ubuntu)
- [Radxa Serial Console Guide](https://wiki.radxa.com/Rockpi4/dev/serial-console)
