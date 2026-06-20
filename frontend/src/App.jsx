import React, { useState, useEffect } from 'react';
import zoopLogo from './assets/images/zoopicon.png';
import { LayoutDashboard, Users, Settings as SettingsIcon, Terminal as LogsIcon } from 'lucide-react';
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
import ClientsList from './components/ClientsList';
import Settings from './components/Settings';
import LogsView from './components/LogsView';

export default function App() {
  const [currentScreen, setCurrentScreen] = useState('dashboard');
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

  // Load initial settings and backend data on mount
  useEffect(() => {
    // 1. Fetch Config
    GetConfig().then((cfg) => {
      setConfig(cfg);
      setTelemetry(prev => ({
        ...prev,
        sharingEnabled: cfg.sharing.enabled
      }));
    }).catch(err => console.error("Failed to load config:", err));

    // 2. Fetch Network Interfaces
    GetInterfaces().then((ifaces) => {
      setInterfaces(ifaces);
    }).catch(err => console.error("Failed to load interfaces:", err));

    // 3. Fetch initial log buffer
    GetLogs().then((initialLogs) => {
      setLogs(initialLogs || []);
    }).catch(err => console.error("Failed to load logs:", err));
  }, []);

  // Set up real-time Wails event listeners for telemetry, clients, and logs
  useEffect(() => {
    // Telemetry tick
    runtime.EventsOn("telemetry_updated", (data) => {
      setTelemetry(data);
    });

    // Connected clients tick
    runtime.EventsOn("clients_updated", (data) => {
      setClients(data || []);
    });

    // Log append event
    runtime.EventsOn("log_added", (newLog) => {
      setLogs((prev) => {
        const updated = [...prev, newLog];
        return updated.slice(-100); // cap size at 100
      });
    });

    // Clean up subscriptions
    return () => {
      runtime.EventsOff("telemetry_updated");
      runtime.EventsOff("clients_updated");
      runtime.EventsOff("log_added");
    };
  }, []);

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
        // Optimistic state update
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

  // Screen layout details mapping
  const screenDetails = {
    dashboard: { title: 'Node Dashboard', subtitle: 'Real-time telemetry and sharing controls' },
    clients: { title: 'Guest Devices', subtitle: 'Manage guest clients routing traffic' },
    settings: { title: 'Daemon Settings', subtitle: 'Configure connection profiles and node properties' },
    logs: { title: 'Diagnostic Logs', subtitle: 'Local console stream and event log history' }
  };

  return (
    <div className="shell">
      {/* Sidebar navigation */}
      <aside className="sidebar">
        <div className="sidebar-brand">
          <img src={zoopLogo} alt="Zoop Logo" />
          <span>Zoop One</span>
          <mark>v1.0.0</mark>
        </div>

        <div className="sidebar-status">
          <span className="status-label">Daemon Status</span>
          <div className={`status-box ${telemetry.sharingEnabled ? 'online' : 'offline'}`}>
            <span className={`status-dot ${telemetry.sharingEnabled ? 'green' : 'red'}`}></span>
            <div className="status-details">
              <strong>{telemetry.sharingEnabled ? 'ONLINE' : 'OFFLINE'}</strong>
              <span>{config ? config.node.displayName : 'Loading...'}</span>
            </div>
          </div>
        </div>

        <nav className="sidebar-nav">
          <div 
            className={`nav-item ${currentScreen === 'dashboard' ? 'active' : ''}`}
            onClick={() => setCurrentScreen('dashboard')}
          >
            <LayoutDashboard />
            <span>Dashboard</span>
          </div>

          <div 
            className={`nav-item ${currentScreen === 'clients' ? 'active' : ''}`}
            onClick={() => setCurrentScreen('clients')}
          >
            <Users />
            <span>Connected Devices</span>
          </div>

          <div 
            className={`nav-item ${currentScreen === 'settings' ? 'active' : ''}`}
            onClick={() => setCurrentScreen('settings')}
          >
            <SettingsIcon />
            <span>Settings</span>
          </div>

          <div 
            className={`nav-item ${currentScreen === 'logs' ? 'active' : ''}`}
            onClick={() => setCurrentScreen('logs')}
          >
            <LogsIcon />
            <span>System Logs</span>
          </div>
        </nav>

        <div className="sidebar-footer">
          <span className="footer-title">Identity Key Hash</span>
          <span className="footer-text" title={config?.node.privateKeyPath}>
            {config ? `...${config.node.privateKeyPath.slice(-24)}` : 'Loading...'}
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
        </header>

        <section className="content">
          {currentScreen === 'dashboard' && (
            <Dashboard 
              telemetry={telemetry} 
              onToggleSharing={handleToggleSharing} 
            />
          )}

          {currentScreen === 'clients' && (
            <ClientsList 
              clients={clients} 
              sharingEnabled={telemetry.sharingEnabled}
              onDisconnectClient={handleDisconnectClient} 
            />
          )}

          {currentScreen === 'settings' && (
            <Settings 
              config={config} 
              interfaces={interfaces} 
              onSaveConfig={handleSaveConfig} 
            />
          )}

          {currentScreen === 'logs' && (
            <LogsView 
              logs={logs} 
            />
          )}
        </section>
      </main>
    </div>
  );
}
