# First-Boot Setup Guide

## Overview

After installing NithronOS, the first-boot process guides you through initial system configuration via a web-based setup wizard. This ensures your system is properly configured and secured before use.

## First-Boot Process

### 1. System Boot

When the system boots for the first time after installation:

1. All services start automatically (nosd, nos-agent, caddy)
2. The system generates a one-time password (OTP)
3. The OTP is displayed on the console after the login prompt appears

### 2. OTP Display

The OTP display includes:
- System IP addresses for web access
- 6-digit one-time password
- Expiration time (30 minutes)

Example console output:
```
╔═══════════════════════════════════════════════════════════════╗
║                  NithronOS First-Boot Setup                   ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Welcome! Please complete the initial setup via web browser:  ║
║                                                                ║
║  Access URLs:                                                  ║
║  - 192.168.1.100                                              ║
║  - 10.0.0.50                                                  ║
║                                                                ║
║  One-Time Password (OTP): 123456                              ║
║                                                                ║
║  This OTP expires in 30 minutes and is required for setup.    ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
```

### 3. Web Access

Open a web browser and navigate to:
- `https://<server-ip>` (HTTPS with self-signed certificate)
- Accept the certificate warning (expected for self-signed certificates)

## Setup Wizard Steps

### Step 1: OTP Verification

- Enter the 6-digit OTP from the console
- The OTP expires after 30 minutes
- Failed attempts are rate-limited

**Troubleshooting**:
- If OTP expired: Wait 10-15 seconds after reboot for new OTP
- If OTP not displayed: Check `journalctl -u nos-firstboot-otp`

### Step 2: Create Administrator Account

**Username Requirements**:
- 3-32 characters
- Lowercase letters, numbers, dash, underscore only
- Must be unique

**Password Requirements**:
- Minimum 12 characters
- Must include: uppercase, lowercase, numbers, symbols
- Password strength meter shows real-time feedback

**Two-Factor Authentication**:
- Optional but recommended
- Can be enabled during setup or later

### Step 3: System Configuration

**Hostname**:
- RFC 1123 compliant
- Used for network identification
- Can include domain (e.g., `nas.home.local`)

**Timezone**:
- Select from list of available timezones
- Affects log timestamps and scheduled tasks
- Default: UTC

**NTP (Network Time Protocol)**:
- Enable for automatic time synchronization
- Recommended for accurate timestamps
- Uses systemd-timesyncd

### Step 4: Network Configuration

**Interface Selection**:
- Choose primary network interface
- Shows current status (up/down)

**Configuration Mode**:
- **DHCP** (recommended): Automatic configuration from router
- **Static**: Manual IP configuration

**Static Configuration**:
- IP Address: CIDR notation (e.g., `192.168.1.100/24`)
- Gateway: Router IP address
- DNS Servers: Primary and optional secondary

**Skip Option**:
- Use current network configuration
- Useful if network is already properly configured

### Step 5: Telemetry Consent

**Privacy-Focused**:
- Completely optional
- Anonymous data only
- No personal information collected

**Data Types** (if enabled):
- System information: OS version, hardware specs
- Usage statistics: Feature usage, performance metrics
- Error reports: Crash logs, error messages

**Privacy Promise**:
- No file names or contents
- No user data
- All data aggregated and anonymous
- Can be changed later in settings

### Step 6: Two-Factor Setup (Optional)

If enabled in Step 2:

**QR Code**:
- Scan with authenticator app (Google Authenticator, Authy, etc.)
- Manual entry code provided as alternative

**Verification**:
- Enter 6-digit code from authenticator
- Confirms TOTP is working correctly

**Recovery Codes**:
- Save these securely
- Each code can be used once
- Required if authenticator is lost

### Step 7: Setup Complete

- Review completion message
- Navigate to sign-in page
- Use created admin credentials

## Post-Setup

### First Login

1. Enter username and password
2. If 2FA enabled: Enter TOTP code
3. Access full dashboard

### Recommended Next Steps

1. **Storage Configuration**:
   - Create storage pools
   - Configure disk monitoring

2. **Network & Remote Access**:
   - Configure firewall rules
   - Set up remote access (VPN/HTTPS)

3. **Shares**:
   - Create SMB/NFS shares
   - Set permissions

4. **Backup**:
   - Configure backup schedules
   - Set up replication targets

## Configuration Files

Key configuration files created during first-boot:

| File | Purpose |
|------|---------|
| `/etc/nos/users.json` | User accounts and roles |
| `/etc/hostname` | System hostname |
| `/etc/timezone` | System timezone |
| `/etc/systemd/network/*.network` | Network configuration |
| `/etc/nos/telemetry/consent.json` | Telemetry settings |

## Modifying Settings

### Change Hostname

```bash
hostnamectl set-hostname new-hostname
```

Or via API:
```bash
curl -X POST https://localhost/api/v1/system/hostname \
  -H "Content-Type: application/json" \
  -d '{"hostname": "new-hostname"}'
```

### Change Timezone

```bash
timedatectl set-timezone America/New_York
```

Or via web UI: Settings → System → Timezone

### Network Reconfiguration

Via web UI: Settings → Network → Interfaces

Or edit `/etc/systemd/network/10-eth0.network`:
```ini
[Match]
Name=eth0

[Network]
DHCP=yes
```

## Troubleshooting

### Cannot Access Web Interface

1. **Check IP address**:
   ```bash
   ip addr show
   ```

2. **Verify services**:
   ```bash
   systemctl status nosd caddy nos-agent
   ```

3. **Check firewall**:
   ```bash
   nft list ruleset | grep 443
   ```

4. **Test locally**:
   ```bash
   curl -k https://localhost
   ```

### OTP Issues

**OTP not displayed**:
```bash
# Check if first-boot is pending
cat /var/lib/nos/firstboot.json

# View OTP directly
cat /var/lib/nos/firstboot-otp

# Check service logs
journalctl -u nos-firstboot-otp
```

**Generate new OTP**:
```bash
# Restart nosd to generate new OTP
systemctl restart nosd
```

### Setup Wizard Errors

**"Backend unreachable"**:
- Check nosd is running: `systemctl status nosd`
- Check API responds: `curl http://127.0.0.1:9000/api/v1/health`

**"Setup already completed"**:
- Normal if setup was previously done
- To reset (WARNING: removes all users):
  ```bash
  rm /etc/nos/users.json
  rm /var/lib/nos/firstboot.json
  systemctl restart nosd
  ```

### Certificate Warnings

Self-signed certificate warnings are expected. To use a trusted certificate:

1. Complete first-boot setup
2. Navigate to Settings → Network → HTTPS
3. Configure Let's Encrypt or upload custom certificate

## Virtual Machine Considerations

### Network Detection

In VMs, multiple network interfaces may be detected:
- Choose the bridged/NAT interface for LAN access
- Avoid host-only interfaces unless intended

### Console Access

For headless VMs:
- Use VM console to view OTP
- Or SSH to VM and run: `cat /var/lib/nos/firstboot-otp`

### Time Sync

Ensure VM time is synchronized:
- Enable NTP in setup wizard
- Or manually: `timedatectl set-ntp true`

## Security Notes

### Default Security

After first-boot:
- No default passwords
- Admin account with strong password
- Optional 2FA protection
- HTTPS enabled (self-signed initially)
- Firewall configured for LAN-only access

### Hardening Recommendations

1. **Enable 2FA** for all admin accounts
2. **Configure HTTPS** with trusted certificate
3. **Review firewall rules** for your environment
4. **Regular updates** via Settings → Updates
5. **Monitor logs** via Dashboard → System Logs

## See Also

- [Installation Guide](installer.md)
- [User Management](../admin/login-and-sessions.md)
- [Network Configuration](../networking.md)
- [Security Model](../security-model.md)
