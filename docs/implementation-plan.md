# GoKrazy NanoPi Zero2 Kernel Implementation Plan

This document outlines the phases required to build a GoKrazy-compatible kernel package for the NanoPi Zero2 (RK3528A SoC).

## Target Hardware

| Component | Specification                           |
| --------- | --------------------------------------- |
| SoC       | Rockchip RK3528A (Quad-core Cortex-A53) |
| RAM       | 1GB/2GB LPDDR4X                         |
| Ethernet  | Native Gigabit (internal GMAC)          |
| eMMC      | Module connector (up to 64GB)           |
| UART      | 1500000 baud, 3.3V                      |

## Mainline Kernel Status

The RK3528A has **active mainline support** (kernel 6.15+). Key drivers:

| Feature                 | Status | Target Version |
| ----------------------- | ------ | -------------- |
| GMAC (Gigabit Ethernet) | Merged | v6.16          |
| eMMC/SD                 | Merged | v6.16          |
| NanoPi Zero2 DTS        | Merged | v6.18          |

---

## Phase 1: Development Environment & Initial Boot

**Goal:** Achieve serial console boot on NanoPi Zero2 with mainline kernel.

- Set up cross-compilation toolchain (aarch64-linux-gnu)
- Clone mainline Linux kernel (v6.18+ or linux-next)
- Build minimal kernel with RK3528 defconfig
- Obtain/build U-Boot for RK3528A
- Create bootable SD card with kernel + DTB
- Verify UART console output at 1500000 baud

**Success Criteria:** Kernel boots to panic (no rootfs) with UART output visible.

---

## Phase 2: Kernel Configuration for Target Features

**Goal:** Configure kernel with all required drivers enabled.

- Start from `defconfig` + RK3528 fragments
- Enable GMAC driver (`CONFIG_DWMAC_ROCKCHIP`)
- Enable eMMC/SD driver (`CONFIG_MMC_DW_ROCKCHIP`)
- Minimize kernel size by disabling unnecessary features
- Test each subsystem individually

**Success Criteria:** `lsblk` shows eMMC; `ip link` shows eth0.

---

## Phase 3: GoKrazy Kernel Package Structure

**Goal:** Package kernel in GoKrazy-compatible format.

- Create `vmlinuz` (compressed kernel image)
- Include device tree blob (`rk3528-nanopi-zero2.dtb`)
- Build required kernel modules into `lib/modules/`
- Create `config.txt` and `cmdline.txt` for boot parameters
- Set up `_build/` directory with kernel config and upstream URL
- Update `Makefile` for reproducible builds

**Success Criteria:** Package structure matches `github.com/gokrazy/kernel`.

---

## Phase 4: Bootloader Integration

**Goal:** Boot GoKrazy from SD card and eMMC.

- Build U-Boot with GoKrazy-compatible boot flow
- Create `boot.scr` for U-Boot script
- Handle SD card boot (mmcblk0)
- Handle eMMC boot (mmcblk2)
- Test boot sequence: U-Boot → kernel → GoKrazy init

**Success Criteria:** GoKrazy boots to init with network and storage visible.

---

## Phase 5: Network & Storage Validation

**Goal:** Verify all target features work under GoKrazy.

- Test Gigabit Ethernet throughput (iperf3)
- Test eMMC read/write performance
- Verify GoKrazy can update itself over network
- Test persistence across reboots

**Success Criteria:** Gigabit Ethernet and eMMC fully operational.

---

## Phase 6: CI/CD & Release

**Goal:** Automated builds and releases.

- Configure GitHub Actions for kernel builds
- Set up reproducible build environment (Docker)
- Automate version tagging
- Create release artifacts
- Document usage instructions in README

**Success Criteria:** Push to main triggers build; releases are downloadable.

---

## Risk Assessment

| Risk                     | Likelihood | Mitigation                                  |
| ------------------------ | ---------- | ------------------------------------------- |
| U-Boot incompatibility   | Medium     | Port from FriendlyElec BSP                  |
| eMMC partition conflicts | Medium     | Use GoKrazy's partition scheme from scratch |
| Performance issues       | Low        | Profile and optimize kernel config          |

## Resources

- [RK3528 Mainline Status](https://github.com/ziyao233/rk3528-mainline)
- [NanoPi Zero2 Wiki](https://wiki.friendlyelec.com/wiki/index.php/NanoPi_Zero2)
- [GoKrazy Kernel Reference](https://github.com/gokrazy/kernel)
- [Armbian Forum Discussion](https://forum.armbian.com/topic/55987-nanopi-zero2-support/)
