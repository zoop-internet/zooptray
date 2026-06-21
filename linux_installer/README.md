# 🐧 Zoop One for Linux

## 📦 Linux Installation Package

This package contains the Linux version of Zoop One with full cross-platform support.

### Included Files

- `zoop-one` - Linux executable (8.8 MB)
- `README.md` - Installation instructions

## 🚀 Installation

### Method 1: Quick Install

```bash
# 1. Download the package
wget https://github.com/Zoop-Network/zooptray/releases/download/v1.0.0-linux/zoop-one-linux.tar.gz

# 2. Extract the files
tar -xzf zoop-one-linux.tar.gz

# 3. Make executable
chmod +x zoop-one

# 4. Run the application
./zoop-one
```

### Method 2: System-wide Installation

```bash
# 1. Download and extract
wget https://github.com/Zoop-Network/zooptray/releases/download/v1.0.0-linux/zoop-one-linux.tar.gz
tar -xzf zoop-one-linux.tar.gz

# 2. Install to /usr/local/bin
sudo mv zoop-one /usr/local/bin/
sudo chown root:root /usr/local/bin/zoop-one
sudo chmod 755 /usr/local/bin/zoop-one

# 3. Run from anywhere
zoop-one
```

## 📋 Features

- ✅ Automatic WireGuard dependency installation
- ✅ System tray integration (requires libayatana-appindicator3)
- ✅ Auto-start configuration
- ✅ Real-time monitoring dashboard
- ✅ Cross-platform WireGuard support

## 🔧 Requirements

### Mandatory
- Linux (Debian/Ubuntu/Fedora/Arch)
- 64-bit system
- Internet connection
- 50MB free disk space

### Recommended
- Debian 11+/Ubuntu 22.04+/Fedora 36+/Arch Linux
- Systemd for auto-start
- libayatana-appindicator3 for system tray

## 🛠️ Setup

### Install Dependencies

```bash
# Debian/Ubuntu
sudo apt update
sudo apt install wireguard wireguard-tools iptables

# Fedora/RHEL
sudo dnf install wireguard-tools iptables

# Arch Linux
sudo pacman -S wireguard-tools iptables
```

### System Tray Support

```bash
# Debian/Ubuntu
sudo apt install libayatana-appindicator3-dev

# Fedora
sudo dnf install libappindicator-gtk3
```

## 🎯 Usage

### Starting Zoop One

```bash
# Basic start
./zoop-one

# Start in background
nohup ./zoop-one > /dev/null 2>&1 &

# System-wide installation
zoop-one
```

### Auto-Start Configuration

```bash
# Create desktop file
mkdir -p ~/.config/autostart
cat > ~/.config/autostart/zoop-one.desktop << EOF
[Desktop Entry]
Type=Application
Name=Zoop One
Comment=Zoop Sharing Node Daemon
Exec=/usr/local/bin/zoop-one
Icon=zoop-one
Terminal=false
Categories=Network;Utility;
X-GNOME-Autostart-enabled=true
EOF
```

## 🔒 Security

- WireGuard encryption for all traffic
- Secure key storage in ~/.zoop/
- Firewall integration with iptables
- Isolated client networking

## 📊 Performance

- Startup time: 1-3 seconds
- Memory usage: 30-80MB
- CPU usage: Low (1-3% typical)
- Bandwidth overhead: Minimal (<3%)

## 🤝 Support

- **Discord**: https://zoopnetwork.com/discord
- **GitHub Issues**: https://github.com/Zoop-Network/zooptray/issues
- **Email**: support@zoopnetwork.com

## 📈 Updates

Check for updates in the About section or download the latest version from GitHub.

---

**Zoop One - Empowering global connectivity through shared resources!** 🌍🚀