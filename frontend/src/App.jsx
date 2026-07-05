import React, { useState, useEffect } from 'react';
import zoopLogo from './assets/images/zoopicon.png';
import { LayoutDashboard, Wifi, Monitor, Wallet, Settings as SettingsIcon, Info, Sun, Moon } from 'lucide-react';
import * as runtime from "../wailsjs/runtime/runtime";
import { useTheme, useConfig, useTelemetry, usePeers, useClients, useLogs, useInterfaces, useSharing } from './hooks';
import { DashboardSkeleton, SettingsSkeleton, DevicesSkeleton } from './components/LoadingSkeleton';

import Dashboard  from './components/Dashboard';
import Sharing    from './components/Sharing';
import Devices    from './components/Devices';
import Earnings   from './components/Earnings';
import Settings   from './components/Settings';
import About      from './components/About';
import ToastContainer from './components/Toast';

const screens = {
  dashboard: { title: 'Dashboard',  subtitle: 'Live overview of your node',     Component: Dashboard },
  sharing:   { title: 'Sharing',    subtitle: 'Control your sharing preferences', Component: Sharing },
  devices:   { title: 'Devices',    subtitle: 'Manage connected guest devices',   Component: Devices },
  earnings:  { title: 'Earnings',   subtitle: 'Track your earnings and rewards',  Component: Earnings },
  settings:  { title: 'Settings',   subtitle: 'Configure your node and network',  Component: Settings },
  about:     { title: 'About',      subtitle: 'Node properties and diagnostic logs', Component: About },
};

const skeletons = {
  dashboard: DashboardSkeleton,
  settings:  SettingsSkeleton,
  devices:   DevicesSkeleton,
};

export default function App() {
  const [theme, toggleTheme] = useTheme();
  const { config, loading: configLoading, eulaVisible, acceptEula, handleSaveConfig } = useConfig();
  const telemetry = useTelemetry();
  const { discoveredPeers, discoveryLoading, connectedPeerId, loadDiscoveredPeers, handleConnectToPeer, handleDisconnectFromPeer } = usePeers();
  const { clients, handleDisconnectClient } = useClients();
  const logs = useLogs();
  const interfaces = useInterfaces();
  const { wgInfo, wgLoading, handleToggleSharing } = useSharing(telemetry.setTelemetry);

  const [currentScreen, setCurrentScreen] = useState('dashboard');
  const [transitioning, setTransitioning] = useState(false);

  const navigate = (screen) => {
    if (screen === currentScreen) return;
    setTransitioning(true);
    setTimeout(() => {
      setCurrentScreen(screen);
      setTransitioning(false);
    }, 100);
  };

  useEffect(() => {
    if (currentScreen === 'devices') {
      loadDiscoveredPeers();
    }
  }, [currentScreen, loadDiscoveredPeers]);

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard size={18} /> },
    { id: 'sharing',   label: 'Sharing',   icon: <Wifi size={18} /> },
    { id: 'devices',   label: 'Devices',   icon: <Monitor size={18} />, badge: clients.length > 0 ? String(clients.length) : null },
    { id: 'earnings',  label: 'Earnings',  icon: <Wallet size={18} />, badgeLabel: 'Soon' },
    { id: 'settings',  label: 'Settings',  icon: <SettingsIcon size={18} /> },
    { id: 'about',     label: 'About',     icon: <Info size={18} /> },
  ];

  const { Component } = screens[currentScreen];
  const Skeleton = skeletons[currentScreen];
  const showSkeleton = configLoading && Skeleton;
  const mergedTelemetry = { ...telemetry.telemetry, sharingEnabled: config?.sharing?.enabled ?? false };

  return (
    <div className="shell">
      <ToastContainer />

      <aside className="sidebar">
        <div className="sidebar-top">
          <div className="sidebar-brand">
            <img src={zoopLogo} alt="Zoop Logo" />
            <span>Zoop One</span>
          </div>

          <nav className="sidebar-nav">
            {navItems.map(item => (
              <div
                key={item.id}
                className={`nav-item ${currentScreen === item.id ? 'active' : ''}`}
                onClick={() => navigate(item.id)}
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

        <div className="sidebar-status-card">
          <div className="sidebar-status-header">
            <div className={`status-dot ${telemetry.telemetry.tunnelActive ? 'green' : 'red'}`}></div>
            <span className="sidebar-status-text">
              {telemetry.telemetry.tunnelActive ? 'Connected' : 'Offline'}
            </span>
          </div>
          <div className="sidebar-status-version">v1.0.0</div>
          <span className="sidebar-status-link" onClick={() => runtime.BrowserOpenURL('https://github.com/Zoop-Network/zooptray/releases')}>
            Check for updates
          </span>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div className="topbar-title">
            <h1>{screens[currentScreen].title}</h1>
            <p>{screens[currentScreen].subtitle}</p>
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
              onClick={toggleTheme}
              title="Toggle theme"
            >
              {theme === 'light' ? <Moon size={17} /> : <Sun size={17} />}
            </button>
          </div>
        </header>

        <section className={`content ${transitioning ? 'content-exit' : 'content-enter'}`}>
          {showSkeleton ? (
            <Skeleton />
          ) : (
            <Component
              telemetry={mergedTelemetry}
              config={config}
              bandwidthHistory={telemetry.bandwidthHistory}
              dataHistory={telemetry.dataHistory}
              uptimeString={telemetry.uptimeString}
              wgInfo={wgInfo}
              wgLoading={wgLoading}
              onToggleSharing={handleToggleSharing}
              clients={clients}
              discoveredPeers={discoveredPeers}
              discoveryLoading={discoveryLoading}
              onDisconnectClient={handleDisconnectClient}
              onRefreshDiscovery={loadDiscoveredPeers}
              onConnectToPeer={handleConnectToPeer}
              onDisconnectFromPeer={handleDisconnectFromPeer}
              connectedPeerId={connectedPeerId}
              interfaces={interfaces}
              onSaveConfig={handleSaveConfig}
              logs={logs}
            />
          )}
        </section>
      </main>

      {eulaVisible && (
        <div className="confirm-overlay" onClick={() => {}}>
          <div className="confirm-dialog" style={{ maxWidth: '720px' }} onClick={e => e.stopPropagation()}>
            <h2>End User License Agreement</h2>
            <div style={{ maxHeight: '48vh', overflow: 'auto', marginBottom: '12px', textAlign: 'left', fontSize: '13.5px', lineHeight: '1.6', color: 'var(--muted)' }}>
              <p style={{ marginBottom: '12px' }}><strong>Zoop Network End User License Agreement</strong></p>
              <p style={{ marginBottom: '10px' }}>By installing or using Zoop One ("Software"), you enter into a binding agreement with Zoop Network on the following terms.</p>
              <p style={{ marginBottom: '10px' }}>1. <strong>Grant of License.</strong> Zoop Network grants you a non-exclusive, non-transferable, revocable license to install and use the Software for personal, non-commercial purposes in accordance with this agreement.</p>
              <p style={{ marginBottom: '10px' }}>2. <strong>Network Participation.</strong> The Software connects to a decentralized peer-to-peer network. You acknowledge that network participation may involve sharing bandwidth and computational resources with other network participants.</p>
              <p style={{ marginBottom: '10px' }}>3. <strong>Privacy.</strong> The Software collects minimal telemetry (connection status, uptime, bandwidth usage) required for network operations. No personal data, browsing history, or payload content is collected, stored, or transmitted to Zoop Network servers.</p>
              <p style={{ marginBottom: '10px' }}>4. <strong>Limitation of Liability.</strong> The Software is provided "as is" without warranty of any kind. Zoop Network shall not be liable for any damages arising from the use or inability to use the Software.</p>
              <p style={{ marginBottom: '10px' }}>5. <strong>Open Source.</strong> The Software is distributed as open source under the terms of the applicable license. You may inspect, modify, and distribute the source code in accordance with that license.</p>
              <p style={{ marginBottom: '10px' }}>6. <strong>Termination.</strong> This agreement terminates automatically if you violate any of its terms. Upon termination, you must uninstall the Software and destroy all copies in your possession.</p>
              <p>By clicking "Accept and Continue," you acknowledge that you have read, understood, and agree to be bound by these terms.</p>
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
              <button className="btn" onClick={() => {}}>Decline</button>
              <button className="btn btn-primary" onClick={acceptEula}>Accept and Continue</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
