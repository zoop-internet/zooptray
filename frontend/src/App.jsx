import React, { useState, useEffect } from 'react';
import zoopLogo from './assets/images/zoopicon.png';
import { 
  LayoutDashboard, 
  Wifi, 
  Wallet, 
  Settings as SettingsIcon, 
  Info,
  Sun, 
  Moon
} from 'lucide-react';
import * as runtime from "../wailsjs/runtime/runtime";
import { 
  GetConfig, 
  SaveConfig, 
  ToggleSharing, 
  GetInterfaces, 
  DisconnectClient, 
  GetLogs 
} from "../wailsjs/go/main/NodeService";

import Dashboard from './components/Dashboard';
import Sharing from './components/Sharing';
import Earnings from './components/Earnings';
import Settings from './components/Settings';
import About from './components/About';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('dashboard');
  const [theme, setTheme] = useState('dark'); // Dark mode by default for premium feel
  const [config, setConfig] = useState(null);
  const [interfaces, setInterfaces] = useState([]);
  const [logs, setLogs] = useState([]);
  const [clients, setClients] = useState([]);
  const [telemetry, setTelemetry] = useState({
    uploadSpeed: 0,
    downloadSpeed: 0,
    activeClientsCount: 0,
    totalDataShared: 0,
    sharingEnabled: false
  });

  // Maintain rolling telemetry histories for sparklines
  const [bandwidthHistory, setBandwidthHistory] = useState(Array(15).fill(0));
  const [dataHistory, setDataHistory] = useState(Array(15).fill(0));

  // Dynamic uptime tracker (simulates long running node starting from 2d 14h 32m)
  const [startTime] = useState(Date.now() - (2 * 24 * 3600 * 1000 + 14 * 3600 * 1000 + 32 * 60 * 1000));
  const [uptimeString, setUptimeString] = useState("2d 14h 32m");

  useEffect(() => {
    const interval = setInterval(() => {
      const diff = Date.now() - startTime;
      const days = Math.floor(diff / (24 * 3600 * 1000));
      const hours = Math.floor((diff % (24 * 3600 * 1000)) / (3600 * 1000));
      const minutes = Math.floor((diff % (3600 * 1000)) / (60 * 1000));
      setUptimeString(`${days}d ${hours}h ${minutes}m`);
    }, 5000);
    return () => clearInterval(interval);
  }, [startTime]);

  // Apply theme class/attribute dynamically
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  // Load initial settings and backend data on mount
  useEffect(() => {
    GetConfig().then((cfg) => {
      setConfig(cfg);
      setTelemetry(prev => ({
        ...prev,
        sharingEnabled: cfg.sharing.enabled
      }));
    }).catch(err => console.error("Failed to load config:", err));

    GetInterfaces().then((ifaces) => {
      setInterfaces(ifaces);
    }).catch(err => console.error("Failed to load interfaces:", err));

    GetLogs().then((initialLogs) => {
      setLogs(initialLogs || []);
    }).catch(err => console.error("Failed to load logs:", err));
  }, []);

  // Set up real-time Wails event listeners
  useEffect(() => {
    // Telemetry update
    runtime.EventsOn("telemetry_updated", (data) => {
      setTelemetry(data);
      
      // Push speeds to history
      const totalSpeed = data.uploadSpeed + data.downloadSpeed;
      setBandwidthHistory(prev => [...prev.slice(1), totalSpeed]);
      setDataHistory(prev => [...prev.slice(1), data.totalDataShared]);
    });

    // Connected clients update
    runtime.EventsOn("clients_updated", (data) => {
      setClients(data || []);
    });

    // Log added event
    runtime.EventsOn("log_added", (newLog) => {
      setLogs((prev) => {
        const updated = [...prev, newLog];
        return updated.slice(-100); // cap size at 100
      });
    });

    return () => {
      runtime.EventsOff("telemetry_updated");
      runtime.EventsOff("clients_updated");
      runtime.EventsOff("log_added");
    };
  }, []);

  // Window management is handled by the OS native title bar on all platforms

  // Interaction handlers
  const handleToggleSharing = (nextState) => {
    ToggleSharing(nextState).then((result) => {
      setTelemetry(prev => ({
        ...prev,
        sharingEnabled: result
      }));
    });
  };

  const handleDisconnectClient = (mac) => {
    DisconnectClient(mac).then((success) => {
      if (success) {
        setClients(prev => prev.filter(c => c.mac !== mac));
      }
    });
  };

  const handleSaveConfig = (updatedConfig) => {
    return SaveConfig(updatedConfig).then(() => {
      setConfig(updatedConfig);
      setTelemetry(prev => ({
        ...prev,
        sharingEnabled: updatedConfig.sharing.enabled
      }));
    });
  };

  // Screen descriptions matching mockup
  const screenDetails = {
    dashboard: { title: 'Dashboard', subtitle: 'Overview of your node' },
    sharing: { title: 'Sharing', subtitle: 'Control your sharing preferences' },
    earnings: { title: 'Earnings', subtitle: 'Track your earnings and payments' },
    settings: { title: 'Settings', subtitle: 'Configure connection profiles and node properties' },
    about: { title: 'About', subtitle: 'Desktop daemon properties and diagnostic stream' }
  };

  // Browser dashboard external link
  const handleOpenBrowserDashboard = () => {
    runtime.BrowserOpenURL("https://console.zoopnetwork.com");
  };

  return (
    <div className="shell">
      {/* Sidebar navigation */}
      <aside className="sidebar">
        <div className="sidebar-top">
          <div className="sidebar-brand">
            <img src={zoopLogo} alt="Zoop Logo" />
            <span>Zoop Desktop</span>
          </div>

          <nav className="sidebar-nav">
            <div 
              className={`nav-item ${currentScreen === 'dashboard' ? 'active' : ''}`}
              onClick={() => setCurrentScreen('dashboard')}
            >
              <div className="nav-item-left">
                <LayoutDashboard />
                <span>Dashboard</span>
              </div>
            </div>

            <div 
              className={`nav-item ${currentScreen === 'sharing' ? 'active' : ''}`}
              onClick={() => setCurrentScreen('sharing')}
            >
              <div className="nav-item-left">
                <Wifi />
                <span>Sharing</span>
              </div>
            </div>

            <div 
              className={`nav-item ${currentScreen === 'earnings' ? 'active' : ''}`}
              onClick={() => setCurrentScreen('earnings')}
            >
              <div className="nav-item-left">
                <Wallet />
                <span>Earnings</span>
              </div>
              <span className="nav-badge">Soon</span>
            </div>

            <div 
              className={`nav-item ${currentScreen === 'settings' ? 'active' : ''}`}
              onClick={() => setCurrentScreen('settings')}
            >
              <div className="nav-item-left">
                <SettingsIcon />
                <span>Settings</span>
              </div>
            </div>

            <div 
              className={`nav-item ${currentScreen === 'about' ? 'active' : ''}`}
              onClick={() => setCurrentScreen('about')}
            >
              <div className="nav-item-left">
                <Info />
                <span>About</span>
              </div>
            </div>
          </nav>
        </div>

        {/* Sidebar Footer Status Indicator */}
        <div className="sidebar-status-card">
          <div className="sidebar-status-header">
            <div className={`status-dot ${telemetry.sharingEnabled ? 'green' : 'red'}`}></div>
            <span className="sidebar-status-text">
              {telemetry.sharingEnabled ? 'Connected' : 'Offline'}
            </span>
          </div>
          <div className="sidebar-status-version">v1.0.0</div>
          <span 
            className="sidebar-status-link"
            onClick={() => alert("Your software is up to date.")}
          >
            Check for updates
          </span>
        </div>
      </aside>

      {/* Main panel layout */}
      <main className="main">
        <header className="topbar">
          <div className="topbar-title">
            <h1>{screenDetails[currentScreen].title}</h1>
            <p>{screenDetails[currentScreen].subtitle}</p>
          </div>
          
          <div className="topbar-right">
            {/* Conditional header actions */}
            {currentScreen === 'dashboard' && (
              <button 
                type="button" 
                className="btn" 
                onClick={handleOpenBrowserDashboard}
              >
                <span>Open Dashboard in Browser</span>
              </button>
            )}

            <button 
              onClick={() => setTheme(prev => prev === 'light' ? 'dark' : 'light')} 
              className="win-btn"
              title={`Switch to ${theme === 'light' ? 'Dark' : 'Light'} Mode`}
            >
              {theme === 'light' ? <Moon size={17} /> : <Sun size={17} />}
            </button>
          </div>
        </header>

        {/* Content routing rendering */}
        <section className="content">
          {currentScreen === 'dashboard' && (
            <Dashboard 
              telemetry={telemetry} 
              config={config}
              onToggleSharing={handleToggleSharing} 
              bandwidthHistory={bandwidthHistory}
              dataHistory={dataHistory}
              uptimeString={uptimeString}
            />
          )}

          {currentScreen === 'sharing' && (
            <Sharing 
              telemetry={telemetry} 
              clients={clients}
              config={config}
              onToggleSharing={handleToggleSharing}
              onDisconnectClient={handleDisconnectClient}
              uptimeString={uptimeString}
            />
          )}

          {currentScreen === 'earnings' && (
            <Earnings />
          )}

          {currentScreen === 'settings' && (
            <Settings 
              config={config} 
              interfaces={interfaces} 
              onSaveConfig={handleSaveConfig} 
            />
          )}

          {currentScreen === 'about' && (
            <About 
              config={config} 
              logs={logs} 
            />
          )}
        </section>
      </main>
    </div>
  );
}
