import React, { useState, useEffect } from 'react';
import zoopLogo from './assets/images/zoopicon.png';
import { 
  LayoutDashboard, Wifi, Monitor, Wallet, Settings as SettingsIcon, Info, Sun, Moon
} from 'lucide-react';
import * as runtime from "../wailsjs/runtime/runtime";
import { GetConfig, SaveConfig, ToggleSharing, GetInterfaces, DisconnectClient, GetLogs, GetTunnelStatus, DiscoverPeers, CreateConnectionProfile } from "../wailsjs/go/main/NodeService";

import Dashboard  from './components/Dashboard';
import Sharing    from './components/Sharing';
import Devices    from './components/Devices';
import Earnings   from './components/Earnings';
import Settings   from './components/Settings';
import About      from './components/About';
import ToastContainer, { toast } from './components/Toast';

export default function App() {
  // Persist theme via localStorage so it survives app restarts
  const [theme, setTheme] = useState(() => localStorage.getItem('zoop-theme') || 'dark');
  const [currentScreen, setCurrentScreen] = useState('dashboard');
  const [config, setConfig]       = useState(null);
  const [interfaces, setInterfaces] = useState([]);
  const [logs, setLogs]           = useState([]);
  const [clients, setClients]     = useState([]);
  const [discoveredPeers, setDiscoveredPeers] = useState([]);
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [telemetry, setTelemetry] = useState({
    uploadSpeed: 0, downloadSpeed: 0,
    activeClientsCount: 0, totalDataShared: 0, sharingEnabled: false,
    tunnelActive: false, tunnelError: '', startedAt: '', publicIP: ''
  });

  const [bandwidthHistory, setBandwidthHistory] = useState(Array(20).fill(0));
  const [dataHistory, setDataHistory]           = useState(Array(20).fill(0));

  // WireGuard tunnel state
  const [wgInfo, setWgInfo]       = useState(null);
  const [wgLoading, setWgLoading] = useState(false);

  const [uptimeString, setUptimeString] = useState('0d 0h 0m');

  // Uptime ticker
  useEffect(() => {
    const updateUptime = () => {
      if (!telemetry.startedAt || !telemetry.tunnelActive) {
        setUptimeString('0d 0h 0m');
        return;
      }
      const startedAt = Date.parse(telemetry.startedAt);
      if (Number.isNaN(startedAt)) {
        setUptimeString('0d 0h 0m');
        return;
      }
      const diff = Math.max(0, Date.now() - startedAt);
      const d = Math.floor(diff / (24 * 3600e3));
      const h = Math.floor((diff % (24 * 3600e3)) / 3600e3);
      const m = Math.floor((diff % 3600e3) / 60e3);
      setUptimeString(`${d}d ${h}h ${m}m`);
    };
    updateUptime();
    const interval = setInterval(updateUptime, 5000);
    return () => clearInterval(interval);
  }, [telemetry.startedAt, telemetry.tunnelActive]);

  // Apply and persist theme
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('zoop-theme', theme);
  }, [theme]);

  // Load initial data
  useEffect(() => {
    GetConfig().then(cfg => {
      setConfig(cfg);
      setTelemetry(prev => ({ ...prev, sharingEnabled: cfg.sharing.enabled }));
    }).catch(err => {
      console.error(err);
      toast.error('Unable to load configuration');
    });

    GetInterfaces().then(setInterfaces).catch(err => {
      console.error(err);
      toast.warning('Unable to detect network interfaces');
    });
    GetLogs().then(l => setLogs(l || [])).catch(err => {
      console.error(err);
      toast.warning('Unable to load logs');
    });
    GetTunnelStatus().then(status => {
      if (status) setTelemetry(prev => ({ ...prev, ...status }));
    }).catch(err => {
      console.error(err);
      toast.warning('Unable to get tunnel status');
    });
  }, []);

  const loadDiscoveredPeers = () => {
    setDiscoveryLoading(true);
    DiscoverPeers()
      .then(setDiscoveredPeers)
      .catch(err => {
        console.error(err);
        setDiscoveredPeers([]);
        toast.warning('Unable to discover network devices.');
      })
      .finally(() => setDiscoveryLoading(false));
  };

  useEffect(() => {
    if (currentScreen === 'devices') {
      loadDiscoveredPeers();
    }
  }, [currentScreen]);

  const handleCopyConnectionProfile = (nodeId) => {
    CreateConnectionProfile(nodeId)
      .then(profile => {
        navigator.clipboard.writeText(profile);
        toast.success('Connection profile copied.');
      })
      .catch(err => {
        console.error(err);
        toast.error('Unable to create connection profile.');
      });
  };

  // Real-time Wails events
  useEffect(() => {
    runtime.EventsOn('telemetry_updated', (data) => {
      setTelemetry(data);
      const totalSpeed = data.uploadSpeed + data.downloadSpeed;
      setBandwidthHistory(prev => data.tunnelActive ? [...prev.slice(1), totalSpeed] : Array(20).fill(0));
      setDataHistory(prev => data.tunnelActive ? [...prev.slice(1), data.totalDataShared] : Array(20).fill(0));
    });
    runtime.EventsOn('clients_updated', (data) => setClients(data || []));
    runtime.EventsOn('log_added', (newLog) => {
      setLogs(prev => [...prev, newLog].slice(-100));
    });

    // WireGuard tunnel lifecycle events
    runtime.EventsOn('wg_tunnel_ready', (info) => {
      setWgLoading(false);
      setWgInfo(info);
    });
    runtime.EventsOn('wg_tunnel_error', (info) => {
      setWgLoading(false);
      setWgInfo(info);
      toast.error(info?.error || 'WireGuard tunnel failed to start');
    });
    runtime.EventsOn('wg_tunnel_stopped', () => {
      setWgLoading(false);
      setWgInfo(null);
    });

    return () => {
      runtime.EventsOff('telemetry_updated');
      runtime.EventsOff('clients_updated');
      runtime.EventsOff('log_added');
      runtime.EventsOff('wg_tunnel_ready');
      runtime.EventsOff('wg_tunnel_error');
      runtime.EventsOff('wg_tunnel_stopped');
    };
  }, []);

  const handleToggleSharing = (nextState) => {
    if (nextState) {
      setWgLoading(true);
      setWgInfo(null);
    } else {
      setWgInfo(null);
      setWgLoading(false);
    }

    ToggleSharing(nextState)
      .then(result => {
        if (!nextState) {
          setTelemetry(prev => ({ ...prev, sharingEnabled: result, tunnelActive: result }));
          toast.info('Sharing disabled successfully.');
        }
      })
      .catch(err => {
        console.error(err);
        setWgLoading(false);
        toast.error('Unable to change sharing state.');
      });
  };

  const handleDisconnectClient = (mac) => {
    DisconnectClient(mac)
      .then(success => {
        if (success) {
          setClients(prev => prev.filter(c => c.mac !== mac));
          toast.success('Client disconnected successfully.');
        } else {
          toast.warning('Client was not found or already disconnected.');
        }
      })
      .catch(err => {
        console.error(err);
        toast.error('Failed to disconnect client.');
      });
  };

  const handleSaveConfig = (updatedConfig) => {
    return SaveConfig(updatedConfig)
      .then(() => {
        setConfig(updatedConfig);
        toast.success('Settings saved successfully');
      })
      .catch(err => {
        console.error(err);
        toast.error(err?.message || 'Failed to save settings');
        throw err;
      });
  };

  const screenDetails = {
    dashboard: { title: 'Dashboard',  subtitle: 'Live overview of your node' },
    sharing:   { title: 'Sharing',    subtitle: 'Control your sharing preferences' },
    devices:   { title: 'Devices',    subtitle: 'Manage connected guest devices' },
    earnings:  { title: 'Earnings',   subtitle: 'Track your earnings and rewards' },
    settings:  { title: 'Settings',   subtitle: 'Configure your node and network' },
    about:     { title: 'About',      subtitle: 'Node properties and diagnostic logs' },
  };

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard size={18} /> },
    { id: 'sharing',   label: 'Sharing',   icon: <Wifi size={18} /> },
    { id: 'devices',   label: 'Devices',   icon: <Monitor size={18} />, badge: clients.length > 0 ? String(clients.length) : null },
    { id: 'earnings',  label: 'Earnings',  icon: <Wallet size={18} />, badgeLabel: 'Soon' },
    { id: 'settings',  label: 'Settings',  icon: <SettingsIcon size={18} /> },
    { id: 'about',     label: 'About',     icon: <Info size={18} /> },
  ];

  return (
    <div className="shell">
      <ToastContainer />

      {/* Sidebar */}
      <aside className="sidebar">
        <div className="sidebar-top">
          <div className="sidebar-brand">
            <img src={zoopLogo} alt="Zoop Logo" />
            <span>Zoop Desktop</span>
          </div>

          <nav className="sidebar-nav">
            {navItems.map(item => (
              <div
                key={item.id}
                className={`nav-item ${currentScreen === item.id ? 'active' : ''}`}
                onClick={() => setCurrentScreen(item.id)}
              >
                <div className="nav-item-left">
                  {item.icon}
                  <span>{item.label}</span>
                </div>
                {item.badgeLabel && (
                  <span className="nav-badge">{item.badgeLabel}</span>
                )}
                {item.badge && (
                  <span className="nav-badge-count">{item.badge}</span>
                )}
              </div>
            ))}
          </nav>
        </div>

        {/* Status Footer */}
        <div className="sidebar-status-card">
          <div className="sidebar-status-header">
            <div className={`status-dot ${telemetry.tunnelActive ? 'green' : 'red'}`}></div>
            <span className="sidebar-status-text">
              {telemetry.tunnelActive ? 'Connected' : 'Offline'}
            </span>
          </div>
          <div className="sidebar-status-version">v1.0.0</div>
          <span className="sidebar-status-link" onClick={() => alert('Your software is up to date.')}>
            Check for updates
          </span>
        </div>
      </aside>

      {/* Main */}
      <main className="main">
        <header className="topbar">
          <div className="topbar-title">
            <h1>{screenDetails[currentScreen].title}</h1>
            <p>{screenDetails[currentScreen].subtitle}</p>
          </div>
          <div className="topbar-right">
            {currentScreen === 'dashboard' && (
              <button type="button" className="btn"
                onClick={() => runtime.BrowserOpenURL('https://console.zoopnetwork.com')}
              >
                Open in Browser
              </button>
            )}
            <button
              className="win-btn"
              onClick={() => setTheme(t => t === 'light' ? 'dark' : 'light')}
              title="Toggle theme"
            >
              {theme === 'light' ? <Moon size={17} /> : <Sun size={17} />}
            </button>
          </div>
        </header>

        <section className="content">
          {currentScreen === 'dashboard' && (
            <Dashboard telemetry={telemetry} config={config}
              onToggleSharing={handleToggleSharing}
              bandwidthHistory={bandwidthHistory} dataHistory={dataHistory}
              uptimeString={uptimeString}
              wgInfo={wgInfo}
              wgLoading={wgLoading}
            />
          )}
          {currentScreen === 'sharing' && (
            <Sharing telemetry={telemetry} config={config}
              onToggleSharing={handleToggleSharing}
              uptimeString={uptimeString}
            />
          )}
          {currentScreen === 'devices' && (
            <Devices
              clients={clients}
              telemetry={telemetry}
              discoveredPeers={discoveredPeers}
              discoveryLoading={discoveryLoading}
              onDisconnectClient={handleDisconnectClient}
              onRefreshDiscovery={loadDiscoveredPeers}
              onCopyConnectionProfile={handleCopyConnectionProfile}
            />
          )}
          {currentScreen === 'earnings' && <Earnings />}
          {currentScreen === 'settings' && (
            <Settings config={config} interfaces={interfaces} onSaveConfig={handleSaveConfig} />
          )}
          {currentScreen === 'about' && (
            <About config={config} logs={logs} />
          )}
        </section>
      </main>
    </div>
  );
}
