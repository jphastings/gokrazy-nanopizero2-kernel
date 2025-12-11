# Phase 2: GoKrazy Integration

**Goal:** Boot a complete GoKrazy system on the NanoPi Zero2.

## Prerequisites

- Phase 1 completed (kernel boots, panics on missing rootfs)
- `gok` tool installed: `go install github.com/gokrazy/tools/cmd/gok@latest`
- Working SD card with U-Boot installed

## Overview

GoKrazy expects a specific partition layout and boot flow:

```
SD Card Layout:
┌─────────────────────────────────────────────────────────────┐
│ Sector 0-63: GPT + reserved                                 │
│ Sector 64+: U-Boot (u-boot-rockchip.bin)                    │
├─────────────────────────────────────────────────────────────┤
│ Partition 1: boot (FAT32, ~256MB)                           │
│   - vmlinuz, *.dtb, boot.scr, cmdline.txt                   │
├─────────────────────────────────────────────────────────────┤
│ Partition 2: root (SquashFS, ~256MB)                        │
│   - GoKrazy root filesystem (read-only)                     │
├─────────────────────────────────────────────────────────────┤
│ Partition 3: perm (ext4, remaining space)                   │
│   - Persistent storage                                      │
└─────────────────────────────────────────────────────────────┘
```

## Step-by-Step Checklist

### 1. Understand GoKrazy Boot Flow

> **Why:** GoKrazy has a specific init system and expects certain kernel features.

GoKrazy boot sequence:

1. U-Boot loads kernel, DTB, and cmdline.txt
2. Kernel boots with `init=/gokrazy/init`
3. GoKrazy init mounts root partition (SquashFS)
4. GoKrazy init starts configured services

Required kernel features:

- SquashFS support (`CONFIG_SQUASHFS=y`)
- ext4 support for perm partition (`CONFIG_EXT4_FS=y`)
- Network drivers for updates
- Loop device support (`CONFIG_BLK_DEV_LOOP=y`)

### 2. Update Kernel Config

- [x] **2.1** Review current kernel config

  ```bash
  # Check if SquashFS is enabled
  grep CONFIG_SQUASHFS cmd/gokr-build-kernel/config.txt
  ```

- [x] **2.2** Add required GoKrazy options to config.txt

  Edit `cmd/gokr-build-kernel/config.txt` and ensure these are present:

  ```text
  # Required for GoKrazy
  CONFIG_SQUASHFS=y
  CONFIG_SQUASHFS_XZ=y
  CONFIG_EXT4_FS=y
  CONFIG_BLK_DEV_LOOP=y

  # Network (already have ethernet, add for updates)
  CONFIG_TUN=y
  CONFIG_NETFILTER=y
  CONFIG_NF_CONNTRACK=y
  CONFIG_NF_NAT=y
  CONFIG_NETFILTER_XT_MATCH_CONNTRACK=y
  CONFIG_IP_NF_IPTABLES=y
  CONFIG_IP_NF_NAT=y
  CONFIG_IP_NF_FILTER=y
  ```

- [x] **2.3** Rebuild kernel

  ```bash
  go run ./cmd/gokr-rebuild-kernel
  ```

### 3. Update Boot Configuration

- [x] **3.1** Fix cmdline.txt for SD card root

  The kernel currently tries to mount `/dev/mmcblk0p2` (eMMC), but we need `/dev/mmcblk1p2` (SD card).

  Edit `cmdline.txt`:

  ```text
  console=ttyS2,1500000 earlycon root=/dev/mmcblk1p2 rootwait ro panic=10 oops=panic init=/gokrazy/init
  ```

  Note the changes:
  - `mmcblk0p2` → `mmcblk1p2` (SD card instead of eMMC)
  - Added `ro` (read-only root)
  - Added `init=/gokrazy/init`

- [x] **3.2** Rebuild boot.scr

  ```bash
  go run ./cmd/gokr-rebuild-uboot
  ```

### 4. Publish Kernel Package Locally

> **Why:** The `gok` tool needs to import the kernel package as a Go module. For development, we use a local `replace` directive.

- [x] **4.1** Ensure kernel package has required files

  The kernel package (this repository) must contain:
  - `vmlinuz` - Compiled kernel image
  - `rk3528-nanopi-zero2.dtb` - Device tree blob
  - `boot.scr` - U-Boot boot script
  - `cmdline.txt` - Kernel command line
  - `kernel.go` - Empty Go package for import
  - `go.mod` - Go module definition

  Verify all files exist:

  ```bash
  ls -la vmlinuz rk3528-nanopi-zero2.dtb boot.scr cmdline.txt kernel.go go.mod
  ```

- [ ] **4.2** Note the local path

  ```bash
  # This is your kernel package path
  echo $PWD
  # Example: /Users/jp/src/personal/gokrazy-nanopizero2-kernel
  ```

### 5. Create GoKrazy Instance

- [x] **5.1** Create a gokrazy instance directory

  ```bash
  mkdir -p ~/gokrazy/nanopi-zero2
  cd ~/gokrazy/nanopi-zero2
  ```

- [x] **5.2** Initialize gokrazy instance

  ```bash
  gok -i nanopi-zero2 new
  ```

  This creates `config.json` in `~/gokrazy/nanopi-zero2/`.

- [ ] **5.3** Edit config.json

  Edit `~/gokrazy/nanopi-zero2/config.json`:

  ```json
  {
    "Hostname": "nanopi-zero2",
    "Packages": [
      "github.com/gokrazy/hello",
      "github.com/gokrazy/serial-busybox"
    ],
    "SerialConsole": "ttyS0,1500000",
    "KernelPackage": "github.com/jphastings/gokrazy-nanopizero2-kernel",
    "FirmwarePackage": "",
    "EEPROMPackage": "",
    "DeviceType": "rock64"
  }
  ```

  > **Note:** Adjust `KernelPackage` to match your module path in `go.mod`.

  There's also a hack here. Once I've completed my work here, I'll submit a patch to have a new DeviceType included included in GoKrazy.

  > **Note:** The `DeviceType` _isn't_ `rock64`, but this this is a hack to have GoKrazy use a [DeviceConfig](https://github.com/gokrazy/internal/blob/c74b4e7749e8ab88b74bec2c3727288f43c47985/deviceconfig/config.go#L52-L62) that expects a uboot partition.

- [x] **5.4** Create go.mod with local replace

  Create `~/gokrazy/nanopi-zero2/go.mod`:

  ```bash
  cd ~/gokrazy/nanopi-zero2
  go mod init nanopi-zero2
  ```

  Then edit `go.mod` to add a replace directive:

  ```go
  module nanopi-zero2

  go 1.21

  replace github.com/jphastings/gokrazy-nanopizero2-kernel => /Users/jp/src/personal/gokrazy-nanopizero2-kernel
  ```

  > Adjust paths to match your actual locations.

### 6. Deploy to SD Card

> **Why:** `gok overwrite` creates the full partition layout and writes all components.

- [x] **6.1** Identify your SD card device

  ```bash
  # macOS
  diskutil list

  # Look for your SD card (e.g., /dev/disk4)
  ```

  **Be extremely careful** - wrong device = data loss!

- [x] **6.2** Deploy with gok overwrite

  ```bash
  cd ~/gokrazy/nanopi-zero2

  # Full overwrite (creates all partitions)
  gok -i nanopi-zero2 overwrite --full /dev/diskX
  ```

  This will:
  - Create the 4-partition layout (MBR, boot, root, perm)
  - Copy kernel, DTB, boot.scr, cmdline.txt to boot partition
  - Build and write SquashFS root filesystem
  - Create empty ext4 perm partition

  > **Note:** Replace `/dev/diskX` with your actual SD card device.

### 7. Test Boot

- [x] **7.1** Connect serial and boot

  ```bash
  picocom -b 1500000 /dev/tty.usbserial-*
  ```

- [x] **7.2** Insert SD card and power on

- [x] **7.3** Verify boot messages

  Expected output:
  ```text
  DDR V1.10
  LPDDR4X, 1056MHz
  ...
  U-Boot 2025.10
  ...
  Loading kernel ...
  Boot args: console=ttyS2,1500000 earlycon root=/dev/mmcblk1p2 rootwait ro panic=10 oops=panic init=/gokrazy/init
  Booting kernel ...

  [    0.000000] Linux version 6.18.0 ...
  [    0.000000] Machine model: FriendlyElec NanoPi Zero2
  ...
  [    x.xxx] VFS: Mounted root (squashfs filesystem) readonly
  ...
  gokrazy: kernel, hardware support, and system health
  gokrazy: eth0: 192.168.x.x
  ```

### 8. Debug Common Issues

#### gok overwrite fails with "unknown kernel package"

```text
cannot resolve KernelPackage: ...
```

**Fix:** Ensure the `replace` directive in `go.mod` is correct and the local path exists.

#### Root filesystem not found

```text
VFS: Cannot open root device "mmcblk1p2"
```

**Fix:** Verify partition exists. The `gok overwrite` partition numbering might differ. Check with U-Boot:

```text
mmc part 1
```

If root is on a different partition, update `cmdline.txt` accordingly.

#### SquashFS mount fails

```text
VFS: Cannot mount root fs of unknown type
```

**Fix:** Kernel missing `CONFIG_SQUASHFS=y`. Rebuild kernel with this option.

#### Init not found

```text
Kernel panic - not syncing: No working init found
```

**Fix:** Root filesystem doesn't have `/gokrazy/init`. This means the SquashFS wasn't built correctly by gok.

#### Network not working

GoKrazy uses DHCP by default. If no IP is assigned:

1. Check kernel has network drivers: `CONFIG_DWMAC_ROCKCHIP=y`
2. Check ethernet cable is connected
3. Check DHCP server is available on network

---

## Alternative: Manual SquashFS Creation

If `gok overwrite` doesn't work with the custom kernel package, you can create a minimal test root filesystem manually.

### Install squashfs-tools

```bash
# macOS (via Homebrew)
brew install squashfs

# Linux
sudo apt install squashfs-tools
```

### Create minimal test init

```bash
# Create root structure
mkdir -p /tmp/gokrazy-root/gokrazy
mkdir -p /tmp/gokrazy-root/proc
mkdir -p /tmp/gokrazy-root/sys
mkdir -p /tmp/gokrazy-root/dev
mkdir -p /tmp/gokrazy-root/tmp

# Download static busybox
curl -L -o /tmp/gokrazy-root/bin/busybox \
  "https://busybox.net/downloads/binaries/1.35.0-x86_64-linux-musl/busybox"
chmod +x /tmp/gokrazy-root/bin/busybox

# Create init script
cat > /tmp/gokrazy-root/gokrazy/init << 'EOF'
#!/bin/busybox sh
echo "=== GoKrazy Test Init ==="
echo "Kernel: $(uname -r)"
echo "Machine: $(uname -m)"

# Mount essential filesystems
/bin/busybox mount -t proc proc /proc
/bin/busybox mount -t sysfs sysfs /sys
/bin/busybox mount -t devtmpfs devtmpfs /dev

echo ""
echo "=== Network Interfaces ==="
/bin/busybox ip link

echo ""
echo "=== Block Devices ==="
/bin/busybox ls -la /dev/mmcblk*

echo ""
echo "=== SUCCESS! NanoPi Zero2 booted with GoKrazy-style init ==="
echo ""
echo "Dropping to shell (type 'poweroff' to shutdown)..."
exec /bin/busybox sh
EOF
chmod +x /tmp/gokrazy-root/gokrazy/init

# Create SquashFS image
mksquashfs /tmp/gokrazy-root /tmp/root.squashfs -noappend -comp xz
```

> **Note:** This requires an ARM64 busybox binary, not x86_64. Download from:
> https://busybox.net/downloads/binaries/ (look for `busybox-armv8l`)

### Write to SD card manually

```bash
# Assuming partition 2 is root
diskutil unmountDisk /dev/diskX
sudo dd if=/tmp/root.squashfs of=/dev/rdiskXs2 bs=1M
sync
```

---

## Phase 2 Completion Criteria

- [x] Kernel config includes SquashFS, ext4, loop device support
- [x] cmdline.txt points to correct SD card partition
- [x] SD card has proper partition layout (boot, root, perm)
- [x] Root partition contains valid SquashFS with GoKrazy init
- [x] System boots to GoKrazy init without panic
- [ ] Network interface (eth0) is visible
- [x] Serial console is functional

---

## Next Steps After Phase 2

Once GoKrazy boots successfully:
- Phase 3: Package this as a proper gokrazy-compatible kernel module
- Phase 4: Polish bootloader (automatic boot.scr detection)
- Phase 5: Test ethernet performance, eMMC boot
- Phase 6: Set up CI/CD for automated builds
