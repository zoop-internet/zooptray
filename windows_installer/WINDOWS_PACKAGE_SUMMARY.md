# 📦 Zoop One Windows Package Summary

## Package Contents

**File**: `zoop-one-windows-installer.zip`
**Size**: 5.5 MB (compressed)
**Platform**: Windows 10/11 (64-bit)
**Version**: 1.0.0 (Cross-Platform Edition)

## Included Files

1. **`zoop-one.exe`** (12.5 MB)
   - Main application executable
   - Built with Wails v2.12.0
   - Cross-platform WireGuard integration
   - System tray support

2. **`README.md`**
   - Quick start guide
   - Basic installation instructions
   - Troubleshooting tips

3. **`INSTALL_WINDOWS.md`**
   - Comprehensive step-by-step installation guide
   - WireGuard setup instructions
   - NAT configuration guide
   - Firewall setup
   - Advanced troubleshooting

## System Requirements

### Minimum Requirements
- **OS**: Windows 10 or 11 (64-bit)
- **CPU**: 1 GHz or faster
- **RAM**: 512 MB
- **Disk Space**: 50 MB
- **Network**: Active internet connection

### Recommended Requirements
- **OS**: Windows 11 (latest updates)
- **CPU**: Dual-core 2 GHz or faster
- **RAM**: 2 GB
- **Disk Space**: 100 MB
- **Network**: Broadband connection

## Dependencies

### Required Software
1. **WireGuard** (included in installer or download from https://www.wireguard.com/install/)
2. **Microsoft Visual C++ Redistributable** (usually already installed)
3. **.NET Framework 4.8** (usually already installed)

### Optional Software
1. **PowerShell 5.1+** (for advanced NAT configuration)
2. **Windows Terminal** (for better command-line experience)

## Installation Steps

### Quick Installation (3 minutes)
1. Extract the ZIP file
2. Install WireGuard from https://www.wireguard.com/install/
3. Double-click `zoop-one.exe`
4. Follow on-screen instructions

### Complete Installation (5-10 minutes)
1. Read `INSTALL_WINDOWS.md` for detailed instructions
2. Install WireGuard
3. Configure firewall settings
4. Set up NAT (optional for best performance)
5. Launch Zoop One
6. Configure your node settings

## Features

### Core Features
- ✅ Cross-platform WireGuard integration
- ✅ Automatic node configuration
- ✅ System tray integration
- ✅ Real-time monitoring dashboard
- ✅ Client management interface
- ✅ Bandwidth tracking
- ✅ Auto-start configuration

### Windows-Specific Features
- ✅ Native Windows system tray icon
- ✅ Windows auto-start support
- ✅ PowerShell NAT configuration scripts
- ✅ Windows Defender firewall compatibility
- ✅ Taskbar integration
- ✅ High DPI display support

## Security

### Built-in Security Features
- **WireGuard Encryption**: Military-grade encryption for all traffic
- **Secure Key Storage**: Cryptographic keys stored in `%APPDATA%\Zoop\`
- **Firewall Integration**: Windows Defender compatible
- **Isolated Networking**: Client traffic separated from host

### Recommended Security Practices
1. **Keep WireGuard updated**
2. **Use strong passwords** for your Zoop account
3. **Monitor connected clients** regularly
4. **Set appropriate bandwidth limits**
5. **Update Zoop One frequently**

## Configuration

### Configuration Files
- **Main Config**: `%APPDATA%\Zoop\config.json`
- **WireGuard Keys**: `%APPDATA%\Zoop\wg_private.key`
- **Logs**: `%APPDATA%\Zoop\zoop.log`

### Configuration Options
```json
{
  "node": {
    "id": "your-node-id",
    "displayName": "My Windows Node",
    "privateKeyPath": "%APPDATA%\\Zoop\\identity.key",
    "isPublic": true
  },
  "sharing": {
    "enabled": true,
    "maxClients": 10,
    "bandwidthCapPercent": 68,
    "routingPolicy": "balanced"
  },
  "system": {
    "autoStartOnBoot": true,
    "preferredInterface": "Wi-Fi"
  }
}
```

## Usage

### Starting Zoop One
```powershell
# Basic start
.\(path)\zoop-one.exe

# Start with admin privileges (recommended)
Start-Process .\zoop-one.exe -Verb RunAs
```

### Managing the Service
```powershell
# Check if running
Get-Process zoop-one

# Stop the application
Stop-Process -Name zoop-one
```

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| "WireGuard not found" | Install WireGuard from official website |
| Application won't start | Run as Administrator |
| No internet for clients | Check firewall and NAT settings |
| Slow performance | Try different network interface |
| Tray icon missing | Check Task Manager, restart app |

### Logs Location
- **Main Log**: `%APPDATA%\Zoop\zoop.log`
- **WireGuard Logs**: Windows Event Viewer → Applications

## Support

### Getting Help
- **Documentation**: https://github.com/zoopnetwork/zoop-one
- **Discord Community**: https://zoopnetwork.com/discord
- **GitHub Issues**: https://github.com/zoopnetwork/zoop-one/issues
- **Email Support**: support@zoopnetwork.com

### Reporting Issues
When reporting issues, please include:
1. Windows version (`winver`)
2. Zoop One version
3. WireGuard version (`wg --version`)
4. Relevant log entries
5. Steps to reproduce

## Updates

### Checking for Updates
1. Open Zoop One
2. Go to About section
3. Click "Check for Updates"

### Manual Update
1. Download latest version
2. Replace `zoop-one.exe`
3. Restart the application

## Uninstallation

### Clean Removal
1. Close Zoop One (right-click tray icon → Quit)
2. Delete installation folder
3. Delete `%APPDATA%\Zoop\` folder
4. Remove startup shortcut (if created)

## Legal

### License
Zoop One is released under the MIT License. See LICENSE file for details.

### Privacy
- No personal data collected without consent
- All traffic encrypted end-to-end
- Compliance with GDPR and CCPA

## Contact

**Zoop Network**
- Website: https://zoopnetwork.com
- Email: support@zoopnetwork.com
- Discord: https://zoopnetwork.com/discord
- GitHub: https://github.com/zoopnetwork

## Changelog

### Version 1.0.0 (Cross-Platform Edition)
- ✅ Added Windows support
- ✅ Cross-platform WireGuard integration
- ✅ Windows auto-start functionality
- ✅ NAT configuration guides
- ✅ Comprehensive documentation
- ✅ Improved error handling

## Credits

- **Wails Framework**: https://wails.io
- **WireGuard**: https://www.wireguard.com
- **React**: https://reactjs.org
- **All contributors**: https://github.com/zoopnetwork/zoop-one/graphs/contributors

---

**Zoop One - Share your connection, empower the world!** 🌍🚀