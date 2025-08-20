# NithronOS ISO (live-build)

This profile builds a Debian Bookworm live ISO and installs local `.deb` artifacts
(nosd, nos-agent, nos-web, and meta `nithronos`) during image creation.

## Build locally
```bash
sudo apt-get update && sudo apt-get install -y live-build xorriso squashfs-tools cpio debootstrap genisoimage
bash packaging/iso/build.sh packaging/iso/local-debs
CI places .deb files in packaging/iso/local-debs before running the script.
Output ISO lands in dist/iso/.

