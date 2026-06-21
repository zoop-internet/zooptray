# 🖥️ Zoop One for Windows

Welcome to Zoop One - the official Windows node daemon for the Zoop Network!

## 📦 Installation Package

This folder contains everything you need to run Zoop One on Windows:

- `zoop-one.exe` - The main application executable
- `README.md` - This installation guide
- `INSTALL_WINDOWS.md` - Step-by-step setup instructions

## 🚀 Quick Start

### 1. Install WireGuard

Zoop One requires WireGuard to be installed on your Windows system:

1. **Download WireGuard**: https://www.wireguard.com/install/
2. **Install the MSI package**: Run the downloaded installer
3. **Restart your computer** (recommended)

### 2. Run Zoop One

Simply double-click `zoop-one.exe` to start the application.

### 3. Configure Your Node

The first time you run Zoop One, it will:
- Generate a cryptographic identity key
- Create a configuration file in `%APPDATA%\Zoop\`
- Set up the system tray icon

## 🛠️ Advanced Setup

### Manual NAT Configuration (Optional)

For best performance, you may want to configure NAT manually:

```powershell
# Run as Administrator
netsh interface portproxy add v4tov4 listenport=51820 listenaddress=0.0.0.0 connectport=51820 connectaddress=10.200.0.1
```

### Firewall Configuration

Ensure your firewall allows:
- UDP port 51820 (WireGuard default)
- The Zoop One executable through Windows Defender

## 🔧 Troubleshooting

### Common Issues

**Issue: "WireGuard not found"**
- Solution: Install WireGuard from the official website

**Issue: "No internet access for clients"**
- Solution: Check NAT configuration and firewall settings

**Issue: "Application won't start"**
- Solution: Run as Administrator (required for network operations)

## 📖 Documentation

For complete documentation, visit our [GitHub repository](https://github.com/zoopnetwork/zoop-one) or check the main README.md file.

## 🤝 Support

Need help? Join our community:
- Discord: [zoopnetwork.com/discord](https://zoopnetwork.com/discord)
- GitHub Issues: [github.com/zoopnetwork/zoop-one/issues](https://github.com/zoopnetwork/zoop-one/issues)

Enjoy sharing your connection with the Zoop Network! 🚀