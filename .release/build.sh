#!/bin/sh
# .release/build.sh <type>

init_image() {
	# Get a fresh void image
	wget -qO void.img.xz http://repo-fastly.voidlinux.org/live/current/void-rpi-aarch64-musl-20240314.img.xz
	xz -d void.img.xz
	dd if=/dev/zero bs=1M count=2000 >> void.img
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
	cp dist/gizmo_linux_arm64/gizmo /mnt/target/usr/local/bin/gizmo
	chmod +x /mnt/target/usr/local/bin/gizmo
	cd /mnt/target || exit 1
	chroot /mnt/target /usr/bin/xbps-install -Suy xbps
	chroot /mnt/target /usr/bin/xbps-install -y xmirror
	chroot /mnt/target /usr/bin/xmirror -s https://repo-fastly.voidlinux.org/current
	chroot /mnt/target /usr/bin/xbps-install -uy
	chroot /mnt/target /usr/bin/chsh -s /bin/bash
}

install_fms() {
	chroot /mnt/target /usr/local/bin/gizmo fms system-install
	chroot /mnt/target /usr/bin/useradd -m -s /bin/bash -c "FMS Admin" -G wheel,storage,dialout,docker admin
	chroot /mnt/target passwd -l root
	echo ENABLE_ROOT_GROWPART=yes > /mnt/target/etc/default/growpart
	echo admin:gizmo | chroot /mnt/target chpasswd -c SHA512
}

install_ds() {
	chroot /mnt/target /usr/local/bin/gizmo ds install
	echo root:gizmo | chpasswd -c SHA512 -R /mnt/target
}

finalize() {
	# Backout, unmount, compress
	rm -rf /mnt/target/var/cache/xbps
	cd - || exit 1
	umount -R /mnt/target
	losetup -d "$LDEV"
	mv void.img "$TYPE.img"
	sync
	zip "$TYPE.zip" "$TYPE.img"
}

TYPE="$1"

init_image
mount_image
install_common

case "$TYPE" in
	fms) install_fms ;;
	driver-station) install_ds ;;
	*) exit 1 ;;
esac

finalize
