# NithronOS ISO (Debian live-build)

## Requirements
- Debian/Ubuntu host (or container)
- live-build (`apt-get install -y live-build`)

## Place local .deb packages (optional)
Put built packages into `config/includes.chroot/root/debs/` or mount at runtime to install `nosd`, `nos-agent`, and `nos-web` during image build.

## Build
```bash
cd packaging/iso/debian
sudo ./auto/config
sudo lb build
```

The resulting ISO will be in the current directory.

On first boot, `nithronos-firstboot.service` will:
- Generate a self-signed TLS cert into `/etc/nos/tls`
- Ensure services (`caddy`, `nftables`, `fail2ban`, `nosd`, `nos-agent`) are enabled
- Print the UI URL and an OTP to the console
