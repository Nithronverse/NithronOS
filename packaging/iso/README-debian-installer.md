# NithronOS Debian Installer

This document describes the Debian Installer integration for NithronOS ISO.

## Components

### 1. Debian Installer Files

The installer uses official Debian bookworm cdrom kernel and initrd (text-capable):
- **Kernel**: `/install.amd/vmlinuz` - Debian installer kernel
- **Initrd**: `/install.amd/initrd.gz` - Debian installer initramfs

These files are fetched by `scripts/fetch-di.sh` during the ISO build process.
The script is idempotent, uses a configurable mirror via `DEBIAN_MIRROR`, and
fails the build if either file is missing.

### 2. Preseed Configuration

The installer uses a preseed file at `/preseed/nithronos.cfg` with:
- **Locale**: en_US.UTF-8
- **Keyboard**: US layout
- **Network**: Auto-configure via DHCP
- **Hostname**: nithronos.local
- **Partitioning**: Entire disk with LVM
- **User**: admin (default password must be changed)
- **Packages**: SSH server and base utilities
- **Services**: Enables nosd, nos-agent, and caddy

### 3. Boot Menu Entries

Both BIOS (ISOLINUX) and UEFI (GRUB) boot menus provide three installer entries designed to avoid blank screens on VMs/KMS:
- **Install NithronOS (Debian Installer - Text)** → text frontend, disables KMS (`fb=false nomodeset`)
- **Install NithronOS (Safe graphics)** → text frontend with `nomodeset` fallback
- **Install NithronOS (Serial console @ttyS0)** → text frontend with serial console
Live entry remains default with a 5s timeout. No `quiet`/`splash` on installer entries.

### 4. Non-blocking Banner

The installer shows a brief ASCII banner via `scripts/banner.sh` that:
- Displays for 3 seconds in the background
- Does not block the installer
- Is triggered by preseed early_command

## Local Testing with QEMU

### Prerequisites

Install QEMU and optionally OVMF for UEFI testing:

```bash
# Debian/Ubuntu
sudo apt-get install qemu-system-x86 ovmf

# Fedora
sudo dnf install qemu-system-x86 edk2-ovmf

# macOS
brew install qemu
```

### Test BIOS Boot

```bash
# Basic test (2GB RAM, serial console)
qemu-system-x86_64 \
  -m 2048 \
  -cdrom dist/iso/NithronOS*.iso \
  -serial stdio \
  -boot d

# With VNC display (connect to localhost:5900)
qemu-system-x86_64 \
  -m 2048 \
  -cdrom dist/iso/NithronOS*.iso \
  -vnc :0 \
  -boot d
```

### Test UEFI Boot

```bash
# With OVMF firmware
qemu-system-x86_64 \
  -m 2048 \
  -cdrom dist/iso/NithronOS*.iso \
  -bios /usr/share/OVMF/OVMF_CODE.fd \
  -serial stdio \
  -boot d
```

### Test Installation

For a full installation test with a virtual disk:

```bash
# Create a 20GB virtual disk
qemu-img create -f qcow2 nithronos.qcow2 20G

# Boot and install
qemu-system-x86_64 \
  -m 2048 \
  -hda nithronos.qcow2 \
  -cdrom dist/iso/NithronOS*.iso \
  -boot d \
  -vnc :0

# After installation, boot from disk
qemu-system-x86_64 \
  -m 2048 \
  -hda nithronos.qcow2 \
  -vnc :0
```

### Automated CI Testing

Run the smoke test to verify the installer boots:

```bash
# Make script executable
chmod +x ci/smoke-di.sh

# Run test (requires built ISO)
./ci/smoke-di.sh dist/iso/NithronOS*.iso
```

The smoke test will:
1. Boot the ISO in QEMU (BIOS mode)
2. Boot the ISO in QEMU (UEFI mode if OVMF available)
3. Check that the Debian Installer starts within 90 seconds
4. Report PASS/FAIL for each boot mode

## Troubleshooting

### Installer Doesn't Start

1. Check that `install.amd/vmlinuz` and `install.amd/initrd.gz` exist in the ISO:
   ```bash
   7z l NithronOS*.iso | grep install.amd
   ```

2. Verify preseed file is present:
   ```bash
   7z l NithronOS*.iso | grep preseed/nithronos.cfg
   ```

3. Check boot parameters in GRUB/ISOLINUX configs include:
   - `preseed/file=/cdrom/preseed/nithronos.cfg`
   - `auto=true priority=critical`

### Banner Blocks Installer

The banner script should run in background via `&`. If it blocks:
1. Check `scripts/banner.sh` exits immediately
2. Verify preseed early_command includes `&` at the end

### Services Not Enabled

The late_command in preseed attempts to:
1. Copy .deb files from ISO to target
2. Install packages with dpkg
3. Enable services with systemctl

Check `/var/log/nithronos-install.log` in the installed system.

## Development

To modify the installer:

1. **Update preseed**: Edit `debian/config/includes.binary/preseed/nithronos.cfg`
2. **Change boot menu**: Edit `debian/config/bootloaders/grub-pc/grub.cfg` (UEFI) or `debian/config/bootloaders/isolinux/txt.cfg` (BIOS)
3. **Modify banner**: Edit `debian/config/includes.binary/scripts/banner.sh`
4. **Rebuild ISO**: Run `packaging/iso/build.sh`

## Security Notes

- The default admin password in preseed is for testing only
- Change the password immediately after installation
- Consider using encrypted preseed for production
- Enable firewall and configure SSH keys post-installation
