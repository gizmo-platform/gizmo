#!/bin/sh
# .release/build.sh <type>

init_image() {
    # Get a fresh void image
    wget -qO void.img.xz http://repo-fastly.voidlinux.org/live/current/void-rpi-aarch64-musl-20250202.img.xz
    xz -d void.img.xz
    dd if=/dev/zero bs=1M count=4000 >> void.img
    growpart void.img 2
    LDEV="$(losetup --show -f -P void.img)"
    resize2fs "${LDEV}p2"
    export LDEV
}

mount_image() {
    # Setup and mount
    mkdir -p /mnt/target
    mount "${LDEV}p2" /mnt/target
    mount "${LDEV}p1" /mnt/target/boot
    for _fs in dev proc sys; do
        mount -o ro --rbind "/$_fs" "/mnt/target/$_fs"
        mount -o ro --make-rslave "/mnt/target/$_fs"
    done
    touch /mnt/target/etc/resolv.conf
    mount -o bind /etc/resolv.conf /mnt/target/etc/resolv.conf
}

install_common() {
    cp dist/gizmo_linux_arm64_v8.0/gizmo /mnt/target/usr/bin/gizmo
    chmod +x /mnt/target/usr/bin/gizmo
    cd /mnt/target || exit 1
    chroot /mnt/target /usr/bin/xbps-install -Suy xbps
    chroot /mnt/target /usr/bin/xbps-install -y xmirror void-repo-nonfree
    chroot /mnt/target /usr/bin/xmirror -s https://repo-fastly.voidlinux.org/current
    chroot /mnt/target /usr/bin/xbps-install -uy
    chroot /mnt/target /usr/bin/chsh -s /bin/bash
    chroot /mnt/target /usr/bin/gizmo version
}

install_fms() {
    chroot /mnt/target /usr/bin/xbps-install -y gizmo-fms

    # Overwrite the packaged version with this version
    cd - || exit 1
    cp dist/gizmo_linux_arm64_v8.0/gizmo /mnt/target/usr/bin/gizmo
    cd /mnt/target || exit 1
    chmod +x /mnt/target/usr/bin/gizmo
    chroot /mnt/target /usr/bin/gizmo fms system-install
    chroot /mnt/target /usr/bin/useradd -m -s /bin/bash -c "FMS Admin" -G wheel,storage,dialout,docker admin
    chroot /mnt/target /usr/bin/ln -sf /var/lib/gizmo/bin/netinstall-cli /usr/bin/netinstall-cli
    chroot /mnt/target /usr/bin/htpasswd -cb /var/lib/gizmo/.htpasswd admin gizmo
    chroot /mnt/target /usr/bin/passwd -l root
    echo ENABLE_ROOT_GROWPART=yes > /mnt/target/etc/default/growpart
    echo admin:gizmo | chroot /mnt/target chpasswd -c SHA512
}

install_ds() {
    chroot /mnt/target /usr/bin/gizmo ds install
    chroot /mnt/target /usr/bin/mkdir -p /usr/share/gizmo/gss/
    chroot /mnt/target /usr/bin/xbps-uhelper fetch 'https://github.com/gizmo-platform/firmware/releases/download/v0.1.8/gss-1_0_R00-v0.1.8.uf2>/usr/share/gizmo/gss/firmware.uf2'
    echo root:gizmo | chpasswd -c SHA512 -R /mnt/target
}

ramdisk() {
    chroot /mnt/target /usr/bin/xbps-install -y dracut parted binutils upx busybox-huge pigz
    chroot /mnt/target /usr/bin/upx /usr/bin/gizmo

    mkdir -p /mnt/target/usr/lib/dracut/modules.d/01gizmo/
    mkdir -p /mnt/target/etc/sv/console/
    cat <<'EOF' > /mnt/target/usr/lib/dracut/modules.d/01gizmo/module-setup.sh
#!/bin/sh
check() {
    return 255
}

installkernel() {
    instmods bridge brcmfmac brcmfmac-wcc af_packet joydev cdc-acm
}

install() {
    inst /etc/group
    inst /etc/passwd
    inst /etc/sv/console/run
    inst /etc/sv/dnsmasq/run
    inst /etc/sv/hostapd/run
    inst /etc/sv/lldpd/run
    inst /etc/sv/nanoklogd/run
    inst /etc/sv/socklog-unix/check
    inst /etc/sv/socklog-unix/log/run
    inst /etc/sv/socklog-unix/run
    inst /etc/sv/udevd/run
    inst /usr/bin/agetty
    inst /usr/bin/dnsmasq
    inst /usr/bin/hostapd
    inst /usr/bin/lldpd
    inst /usr/bin/lldpcli
    inst /usr/bin/nanoklogd
    inst /usr/bin/runsv
    inst /usr/bin/runsvdir
    inst /usr/bin/socklog
    inst /usr/bin/socklog-check
    inst /usr/bin/sv
    inst /usr/bin/syslog-stripdate
    inst /usr/bin/tryto
    inst /usr/bin/uncat
    inst /usr/bin/vlogger
    inst /usr/bin/gizmo
    inst /usr/share/gizmo/gss/firmware.uf2
    inst /usr/local/bin/gss-loader
    inst /var/log/socklog/everything/config

    inst_rules /etc/udev/rules.d/50-gizmo.rules
    inst_hook pre-mount 01 "$moddir/gizmo.sh"
    inst_hook cmdline 99 "$moddir/parse-gizmo-root.sh"
}
EOF

    cat <<EOF > /mnt/target/usr/lib/dracut/modules.d/01gizmo/parse-gizmo-root.sh
#!/bin/sh
if [ "${root}"="gizmo-ramdisk" ] ; then
   rootok=1
fi
EOF

    cat <<EOF > /mnt/target/usr/lib/dracut/modules.d/01gizmo/gizmo.sh
#!/bin/sh
export GIZMO_BOOTMODE=RAMDISK
/usr/bin/mkdir -p /etc/runit/runsvdir/default
/usr/bin/ln -s /etc/sv/console /etc/runit/runsvdir/default/
/usr/bin/ln -s /etc/sv/udevd /etc/runit/runsvdir/default/
/usr/bin/ln -s /etc/runit/runsvdir/default /var/service
/usr/bin/modprobe bridge
/usr/bin/udevd --daemon
/usr/bin/udevadm trigger --action=add --type=subsystems
/usr/bin/udevadm trigger --action=add --type=devices
/usr/bin/udevadm settle
/usr/bin/mkdir /boot
/usr/bin/gizmo ds gss-autoconf || /usr/bin/mount -o ro /dev/mmcblk0p1 /boot
/usr/bin/gizmo ds configure /boot/gsscfg.json
/usr/bin/sysctl -p /etc/sysctl.conf
/usr/bin/ip link set lo up
/usr/bin/gizmo version
exec /usr/bin/runsvdir /etc/runit/runsvdir/default
EOF

    cat <<EOF > /mnt/target/boot/config.txt
gpu_mem=16
disable_poe_fan=1
initramfs initramfs followkernel
EOF

    cat <<EOF > /mnt/target/boot/cmdline.txt
root=gizmo-ramdisk initrd=initramfs console=tty1 dwc_otg.lpm_enable=0 elevator=noop
EOF

    cat <<EOF > /mnt/target/etc/sv/console/run
#!/bin/sh
cd /
exec /usr/bin/chpst -P /usr/bin/agetty --autologin root --login-program /usr/bin/sh --login-options "" tty2 38400 linux
EOF
    chmod +x /mnt/target/etc/sv/console/run

    kver=$(ls /mnt/target/usr/lib/modules/)
    chroot /mnt/target /usr/bin/dracut --kver "$kver" --modules "gizmo busybox kernel-modules base" /boot/initramfs

    mkdir /mnt/target/tmp/ramds
    for _f in initramfs kernel8.img fixup_cd.dat start_cd.elf bootcode.bin bcm2710-rpi-zero-2-w.dtb config.txt cmdline.txt; do
        cp /mnt/target/boot/$_f /mnt/target/tmp/ramds/
    done

    zip -jr "$outdir/ds-ramdisk.zip" /mnt/target/tmp/ramds/*
}

backout() {
    # Backout, unmount, compress
    rm -rf /mnt/target/var/cache/xbps
    cd - || exit 1
    umount -R /mnt/target
    losetup -d "$LDEV"
}

finalize() {
    mv void.img "$TYPE.img"
    sync
    zip "$TYPE.zip" "$TYPE.img"
}

TYPE="$1"

outdir=$(pwd)

init_image
mount_image
install_common

case "$TYPE" in
    fms) install_fms ;;
    driver-station) install_ds ;;
    ds-ramdisk)
        install_ds
        ramdisk
        ;;
    *) exit 1 ;;
esac

backout

case "$TYPE" in
    fms) finalize ;;
    driver-station) finalize ;;
esac
