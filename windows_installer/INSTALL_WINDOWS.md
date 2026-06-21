# 📋 Step-by-Step Windows Installation Guide

## Prerequisites

Before installing Zoop One, ensure your Windows system meets these requirements:

- **Windows 10 or 11** (64-bit)
- **Administrator privileges** (for network configuration)
- **Internet connection** (for WireGuard download)
- **At least 50MB free disk space**

## Step 1: Install WireGuard

Zoop One uses WireGuard for secure tunneling. You need to install it first:

### Method A: Automatic Installation (Recommended)

1. **Download WireGuard**:
   - Visit https://www.wireguard.com/install/
   - Click "Windows Installer" to download the MSI package

2. **Run the installer**:
   - Double-click the downloaded `WireGuard-Installer.msi` file
   - Follow the installation wizard
   - Click "Finish" when complete

3. **Verify installation**:
   - Open Command Prompt
   - Type: `wg --version`
   - You should see WireGuard version information

### Method B: Manual Installation

If the automatic method doesn't work:

1. Download from: https://download.wireguard.com/windows-client/wireguard-installer/
2. Choose the latest version (e.g., `WireGuard-Installer-0.5.3.msi`)
3. Run the MSI file as Administrator

## Step 2: Install Zoop One

1. **Extract the package**:
   - If you downloaded a ZIP file, extract it to a folder of your choice
   - If you have the EXE file, you can run it directly

2. **Run Zoop One**:
   - Double-click `zoop-one.exe`
   - Click "Yes" if prompted by User Account Control

3. **First-time setup**:
   - The application will create a configuration folder in `%APPDATA%\Zoop\`
   - A system tray icon will appear
   - Your node identity will be generated automatically

## Step 3: Configure Your Node

1. **Open the application interface**:
   - Right-click the system tray icon
   - Click "Show" or "Open"

2. **Basic configuration**:
   - **Node Name**: Give your node a friendly name
   - **Visibility**: Choose "Public" or "Private"
   - **Bandwidth**: Set your sharing limits
   - **Auto-start**: Enable if you want Zoop One to launch with Windows

3. **Advanced settings**:
   - **Network Interface**: Select your primary network adapter
   - **Port**: Default is 51820 (WireGuard standard)
   - **Client Limits**: Set maximum concurrent clients

## Step 4: Start Sharing

1. **Enable sharing**:
   - Click the "Start Sharing" button
   - Wait for the tunnel to initialize (may take 10-30 seconds)

2. **Verify connection**:
   - Check the dashboard for tunnel status
   - Look for "Connected" status in the system tray

3. **Monitor activity**:
   - View connected clients in the "Devices" tab
   - Monitor bandwidth usage in the "Dashboard"

## Step 5: Auto-Start Configuration (Optional)

To have Zoop One start automatically with Windows:

1. **Manual method**:
   - Press `Win + R`, type `shell:startup`, and press Enter
   - Create a shortcut to `zoop-one.exe` in this folder

2. **Through the app**:
   - Go to Settings
   - Enable "Auto-start on boot"
   - Click "Save"

## Step 6: Firewall Configuration

Ensure your firewall allows Zoop One to operate:

1. **Windows Defender Firewall**:
   - Open Windows Security
   - Go to "Firewall & network protection"
   - Click "Allow an app through firewall"
   - Add `zoop-one.exe` to the allowed list

2. **Port forwarding** (if behind NAT):
   - Forward UDP port 51820 to your computer's local IP
   - Consult your router's documentation for specific instructions

## Step 7: NAT Configuration (Advanced Users)

For optimal performance, configure NAT manually:

1. **Open PowerShell as Administrator**
2. **Run these commands**:

```powershell
# Add NAT rule
netsh interface portproxy add v4tov4 listenport=51820 listenaddress=0.0.0.0 connectport=51820 connectaddress=10.200.0.1

# Verify the rule
netsh interface portproxy show all
```

3. **To remove the rule later**:

```powershell
netsh interface portproxy delete v4tov4 listenport=51820 listenaddress=0.0.0.0
```

## Troubleshooting

### Common Issues and Solutions

**Problem: "WireGuard not found" error**
- **Solution**: Install WireGuard from the official website and restart your computer

**Problem: Application crashes on startup**
- **Solution**: Run as Administrator (right-click → Run as administrator)

**Problem: No internet access for connected clients**
- **Solution**: Check firewall settings and NAT configuration

**Problem: Slow connection speeds**
- **Solution**: Try a different network interface or adjust bandwidth limits

**Problem: System tray icon doesn't appear**
- **Solution**: Check Task Manager for running processes and restart the app

### Logs and Diagnostics

To access logs for troubleshooting:

1. Open the application
2. Go to the "About" section
3. Click "View Logs"
4. Look for error messages or warnings

Logs are also stored in: `%APPDATA%\Zoop\zoop.log`

## Uninstallation

To completely remove Zoop One:

1. **Close the application** (right-click tray icon → Quit)
2. **Delete the installation folder**
3. **Remove configuration files**:
   - Delete `%APPDATA%\Zoop\` folder
4. **Remove auto-start entry** (if configured):
   - Delete the shortcut from the Startup folder

## Support

Need help? Contact us:

- **Discord**: [zoopnetwork.com/discord](https://zoopnetwork.com/discord)
- **GitHub Issues**: [github.com/zoopnetwork/zoop-one/issues](https://github.com/zoopnetwork/zoop-one/issues)
- **Email**: support@zoopnetwork.com

## Next Steps

Now that you're set up:

1. 📊 **Monitor your node**: Check the dashboard regularly
2. 🔄 **Update frequently**: Get the latest features and security fixes
3. 🤝 **Join the community**: Share your experience and get support
4. 💡 **Provide feedback**: Help us improve Zoop One

Welcome to the Zoop Network! You're now part of a global community sharing internet access. 🌍🚀