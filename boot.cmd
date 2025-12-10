echo "Loading kernel ..."

# Load compressed kernel image
load ${devtype} ${devnum}:${bootpart} ${kernel_addr_r} vmlinuz

# Load cmdline.txt into memory and use as bootargs
load ${devtype} ${devnum}:${bootpart} ${ramdisk_addr_r} cmdline.txt
setexpr cmdline_end ${ramdisk_addr_r} + ${filesize}
mw.b ${cmdline_end} 0 1
setexpr.s bootargs *${ramdisk_addr_r}

echo "Boot args: ${bootargs}"

# Load device tree
setenv fdtfile rk3528-nanopi-zero2.dtb
load ${devtype} ${devnum}:${bootpart} ${fdt_addr_r} ${fdtfile}
fdt addr ${fdt_addr_r}

echo "Booting kernel ..."

# Boot without initrd
booti ${kernel_addr_r} - ${fdt_addr_r}
