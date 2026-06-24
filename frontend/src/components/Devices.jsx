import React, { useState } from 'react';
import { Monitor, Smartphone, Laptop, Tablet, Wifi, XCircle, Search, RefreshCw, Copy } from 'lucide-react';
import { toast } from './Toast';

// Pick an icon based on device name heuristics
function DeviceIcon({ name }) {
  const n = (name || '').toLowerCase();
  if (n.includes('iphone') || n.includes('pixel') || n.includes('android') || n.includes('samsung')) 
    return <Smartphone size={18} />;
  if (n.includes('ipad') || n.includes('tablet')) 
    return <Tablet size={18} />;
  if (n.includes('macbook') || n.includes('thinkpad') || n.includes('laptop') || n.includes('desktop'))
    return <Laptop size={18} />;
  return <Monitor size={18} />;
}

export default function Devices({ clients, telemetry, discoveredPeers, discoveryLoading, onDisconnectClient, onRefreshDiscovery, onCopyConnectionProfile }) {
  const [search, setSearch] = useState('');
  const [confirmKick, setConfirmKick] = useState(null); // mac address to confirm kick

  const isOnline = telemetry.tunnelActive;

  const filtered = clients.filter(c =>
    c.name.toLowerCase().includes(search.toLowerCase()) ||
    c.ip.toLowerCase().includes(search.toLowerCase()) ||
    c.mac.toLowerCase().includes(search.toLowerCase())
  );

  const filteredPeers = discoveredPeers.filter(peer =>
    peer.displayName.toLowerCase().includes(search.toLowerCase()) ||
    peer.endpoint.toLowerCase().includes(search.toLowerCase()) ||
    peer.nodeId.toLowerCase().includes(search.toLowerCase()) ||
    peer.region.toLowerCase().includes(search.toLowerCase())
  );

  const handleKickRequest = (client) => {
    setConfirmKick(client);
  };

  const handleKickConfirm = () => {
    if (!confirmKick) return;
    onDisconnectClient(confirmKick.mac);
    toast.warning(`${confirmKick.name} has been disconnected.`);
    setConfirmKick(null);
  };

  const handleCopyProfile = (peer) => {
    onCopyConnectionProfile(peer.nodeId);
  };

  return (
    <div className="devices-layout">
      {/* Header bar */}
      <div className="devices-header-bar">
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span className="devices-count-badge">
              {clients.length} connected
            </span>
            <span style={{ fontSize: '13px', color: 'var(--muted)' }}>
              {isOnline ? 'Node is online — routing active' : 'Node is offline'}
            </span>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
          <button className="btn btn-secondary" type="button" onClick={onRefreshDiscovery}>
            <RefreshCw size={14} />
            <span style={{ marginLeft: '6px' }}>Refresh Network</span>
          </button>
          <div className="devices-search">
            <Search size={15} />
            <input
              type="text"
              placeholder="Search devices..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="devices-search-input"
            />
          </div>
        </div>
      </div>

      {/* Connected clients section */}
      {!isOnline ? (
        <div className="empty-state">
          <Wifi size={36} style={{ color: 'var(--muted-dark)' }} />
          <h3>Node is Offline</h3>
          <p>Go to <strong>Sharing</strong> and enable your node to start routing guest device traffic.</p>
        </div>
      ) : clients.length === 0 ? (
        <div className="empty-state">
          <Monitor size={36} style={{ color: 'var(--muted-dark)' }} />
          <h3>No Devices Connected</h3>
          <p>Waiting for nearby devices to discover your sharing node…</p>
        </div>
      ) : filtered.length === 0 ? (
        <div className="empty-state">
          <Search size={36} style={{ color: 'var(--muted-dark)' }} />
          <h3>No Results for "{search}"</h3>
          <p>Try searching by device name, IP address, or MAC.</p>
        </div>
      ) : (
        <div className="devices-grid">
          {filtered.map(client => (
            <div key={client.mac} className="device-card">
              <div className="device-card-top">
                <div className="device-icon-wrap">
                  <DeviceIcon name={client.name} />
                </div>
                <div className="device-info">
                  <div className="device-name">{client.name}</div>
                  <div className="device-mac">{client.mac}</div>
                </div>
                <div className="device-status-dot green"></div>
              </div>

              <div className="device-stats">
                <div className="device-stat">
                  <span className="device-stat-label">IP Address</span>
                  <span className="device-stat-value mono">{client.ip}</span>
                </div>
                <div className="device-stat">
                  <span className="device-stat-label">Speed</span>
                  <span className="device-stat-value speed">{client.speed.toFixed(1)} Mbps</span>
                </div>
                <div className="device-stat">
                  <span className="device-stat-label">Data Routed</span>
                  <span className="device-stat-value mono">{client.dataShared.toFixed(1)} MB</span>
                </div>
                <div className="device-stat">
                  <span className="device-stat-label">Connected Since</span>
                  <span className="device-stat-value mono">{client.connectedAt}</span>
                </div>
              </div>

              <button
                type="button"
                className="btn btn-danger device-kick-btn"
                onClick={() => handleKickRequest(client)}
              >
                <XCircle size={14} />
                <span>Disconnect</span>
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Discovered peers section — always visible regardless of local tunnel state */}
      {discoveredPeers && discoveredPeers.length > 0 && (
        <div className="discovery-section">
          <h3>Discovered Network Nodes</h3>
          <div className="devices-grid">
            {filteredPeers.map(peer => (
              <div key={peer.nodeId} className="device-card">
                <div className="device-card-top">
                  <div className="device-icon-wrap">
                    <DeviceIcon name={peer.displayName} />
                  </div>
                  <div className="device-info">
                    <div className="device-name">{peer.displayName || peer.nodeId}</div>
                    <div className="device-mac">{peer.nodeId}</div>
                  </div>
                  <div className="device-status-dot green"></div>
                </div>

                <div className="device-stats">
                  <div className="device-stat">
                    <span className="device-stat-label">Endpoint</span>
                    <span className="device-stat-value mono">{peer.endpoint}</span>
                  </div>
                  <div className="device-stat">
                    <span className="device-stat-label">Region</span>
                    <span className="device-stat-value">{peer.region || 'global'}</span>
                  </div>
                  <div className="device-stat">
                    <span className="device-stat-label">Public</span>
                    <span className="device-stat-value">{peer.isPublic ? 'Yes' : 'No'}</span>
                  </div>
                </div>

                <button
                  type="button"
                  className="btn btn-primary device-kick-btn"
                  onClick={() => handleCopyProfile(peer)}
                >
                  <Copy size={14} />
                  <span>Copy Client Profile</span>
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Kick Confirmation Dialog */}
      {confirmKick && (
        <div className="confirm-overlay" onClick={() => setConfirmKick(null)}>
          <div className="confirm-dialog" onClick={e => e.stopPropagation()}>
            <div className="confirm-icon">
              <XCircle size={28} />
            </div>
            <h3>Disconnect Device?</h3>
            <p>
              <strong>{confirmKick.name}</strong> ({confirmKick.ip}) will be immediately 
              disconnected from your sharing node.
            </p>
            <div className="confirm-actions">
              <button 
                type="button" 
                className="btn" 
                onClick={() => setConfirmKick(null)}
              >
                Cancel
              </button>
              <button 
                type="button" 
                className="btn btn-danger" 
                onClick={handleKickConfirm}
              >
                Yes, Disconnect
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
