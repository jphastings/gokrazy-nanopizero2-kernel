# Phase 2: GoKrazy Integration

**Goal:** Boot a complete GoKrazy system on the NanoPi Zero2.

## Prerequisites

- Phase 1 completed (kernel boots, panics on missing rootfs)
- `gokr-packer` installed: `go install github.com/gokrazy/tools/cmd/gokr-packer@latest`
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

- [ ] **2.1** Review current kernel config

  ```bash
  # Check if SquashFS is enabled
  grep CONFIG_SQUASHFS cmd/gokr-build-kernel/config.txt
  ```

- [ ] **2.2** Add required GoKrazy options to config.txt

  Edit `cmd/gokr-build-kernel/config.txt` and ensure these are present:

  ```
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

- [ ] **2.3** Rebuild kernel

  ```bash
  go run ./cmd/gokr-rebuild-kernel
  ```

### 3. Update Boot Configuration

- [ ] **3.1** Fix cmdline.txt for SD card root

  The kernel currently tries to mount `/dev/mmcblk0p2` (eMMC), but we need `/dev/mmcblk1p2` (SD card).

  Edit `cmdline.txt`:

  ```
  console=ttyS2,1500000 earlycon root=/dev/mmcblk1p2 rootwait ro panic=10 oops=panic init=/gokrazy/init
  ```

  Note the changes:
  - `mmcblk0p2` → `mmcblk1p2` (SD card instead of eMMC)
  - Added `ro` (read-only root)
  - Added `init=/gokrazy/init`

- [ ] **3.2** Rebuild boot.scr

  ```bash
  go run ./cmd/gokr-rebuild-uboot
  ```

### 4. Create GoKrazy Instance Configuration

- [ ] **4.1** Create a gokrazy instance directory

  ```bash
  mkdir -p ~/gokrazy/nanopi-zero2
  cd ~/gokrazy/nanopi-zero2
  ```

- [ ] **4.2** Initialize gokrazy config

  ```bash
  gok new
  ```

- [ ] **4.3** Edit config.json

  Create/edit `config.json`:

  ```json
  {
    "Hostname": "nanopi-zero2",
    "Packages": [
      "github.com/gokrazy/hello",
      "github.com/gokrazy/serial-busybox"
    ],
    "SerialConsole": "ttyS2,1500000",
    "KernelPackage": "",
    "FirmwarePackage": "",
    "EEPROMPackage": ""
  }
  ```

  > Note: We'll specify custom kernel/firmware via command line flags.

### 5. Build GoKrazy Image

- [ ] **5.1** Build root filesystem

  For initial testing, we'll create the image manually since gokr-packer doesn't know about our custom board yet.

  ```bash
  # Build a root filesystem image
  gok build -o /tmp/gokrazy-root.squashfs
  ```

- [ ] **5.2** Alternative: Create minimal test rootfs

  If gokr-packer integration is complex, create a minimal SquashFS for testing:

  ```bash
  # Create minimal init for testing
  mkdir -p /tmp/gokrazy-root/gokrazy

  # Create a simple init script that just prints and sleeps
  cat > /tmp/gokrazy-root/gokrazy/init << 'EOF'
  #!/bin/sh
  echo "GoKrazy init starting..."
  echo "Kernel: $(uname -r)"
  echo "Success! GoKrazy booted on NanoPi Zero2"

  # Mount essential filesystems
  mount -t proc proc /proc
  mount -t sysfs sysfs /sys
  mount -t devtmpfs devtmpfs /dev

  # Show network interfaces
  ip link

  # Keep system running
  echo "Dropping to shell..."
  exec /bin/sh
  EOF
  chmod +x /tmp/gokrazy-root/gokrazy/init

  # Need busybox for basic commands
  # (This is just for testing - real GoKrazy uses Go binaries)
  ```

  > This step requires more research into GoKrazy's actual init requirements.

### 6. Prepare SD Card with Full Layout

- [ ] **6.1** Create proper partition layout

  On macOS, use `diskutil` carefully or use a Linux VM for better control:

  ```bash
  # This is approximate - may need adjustment
  diskutil partitionDisk /dev/diskX GPT \
    FAT32 BOOT 256MB \
    FAT32 ROOT 512MB \
    FAT32 PERM 0b
  ```

  > Note: ROOT should be SquashFS but macOS can't create that. We'll dd the image.

- [ ] **6.2** Write U-Boot

  ```bash
  diskutil unmountDisk /dev/diskX
  sudo dd if=u-boot-rockchip.bin of=/dev/rdiskX seek=64 bs=512
  ```

- [ ] **6.3** Copy boot files

  ```bash
  diskutil mount /dev/diskXs1  # or wait for auto-mount
  cp vmlinuz /Volumes/BOOT/
  cp rk3528-nanopi-zero2.dtb /Volumes/BOOT/
  cp boot.scr /Volumes/BOOT/
  cp cmdline.txt /Volumes/BOOT/
  diskutil unmount /Volumes/BOOT
  ```

- [ ] **6.4** Write root filesystem

  ```bash
  # This requires the SquashFS image from step 5
  diskutil unmountDisk /dev/diskX
  sudo dd if=/tmp/gokrazy-root.squashfs of=/dev/rdiskXs2 bs=1M
  ```

### 7. Test Boot

- [ ] **7.1** Connect serial and boot

  ```bash
  picocom -b 1500000 /dev/tty.usbserial-*
  ```

- [ ] **7.2** Insert SD card and power on

- [ ] **7.3** Verify boot messages

  Expected output:
  ```
  [    0.000000] Linux version 6.18.0 ...
  [    0.000000] Machine model: FriendlyElec NanoPi Zero2
  ...
  [    x.xxx] VFS: Mounted root (squashfs filesystem) readonly
  ...
  GoKrazy init starting...
  ```

### 8. Debug Common Issues

#### Root filesystem not found

```
VFS: Cannot open root device "mmcblk1p2"
```

**Fix:** Verify partition exists and cmdline.txt has correct device.

At U-Boot prompt:
```
mmc part 1
```

#### SquashFS mount fails

```
VFS: Cannot mount root fs of unknown type
```

**Fix:** Kernel missing `CONFIG_SQUASHFS=y`. Rebuild kernel.

#### Init not found

```
Kernel panic - not syncing: No working init found
```

**Fix:** Root filesystem doesn't have `/gokrazy/init`. Check SquashFS contents.

---

## Phase 2 Completion Criteria

- [ ] Kernel config includes SquashFS, ext4, loop device support
- [ ] cmdline.txt points to correct SD card partition
- [ ] SD card has 3-partition layout (boot, root, perm)
- [ ] Root partition contains valid SquashFS with GoKrazy init
- [ ] System boots to GoKrazy init without panic
- [ ] Network interface (eth0) is visible
- [ ] Serial console is functional

---

## Research Needed

1. **GoKrazy init requirements:** What does `/gokrazy/init` actually need?
   - Review https://github.com/gokrazy/gokrazy/tree/main/cmd/init

2. **gokr-packer integration:** Can we use `gokr-packer -overwrite` with custom kernel?
   - Need to create a Go package that exports kernel/dtb/firmware

3. **Kernel module handling:** Does GoKrazy need any kernel modules, or should everything be built-in?

4. **Network configuration:** How does GoKrazy configure eth0? DHCP? Static?

---

## Next Steps After Phase 2

Once GoKrazy boots successfully:
- Phase 3: Package this as a proper gokrazy-compatible kernel module
- Phase 4: Polish bootloader (automatic boot.scr detection)
- Phase 5: Test ethernet performance, eMMC boot
- Phase 6: Set up CI/CD for automated builds
