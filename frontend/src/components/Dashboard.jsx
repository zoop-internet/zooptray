import React from 'react';
import { Power, ArrowDown, ArrowUp, Users, HardDrive } from 'lucide-react';

export default function Dashboard({ telemetry, onToggleSharing }) {
  const isOnline = telemetry.sharingEnabled;

  return (
    <div>
      {/* Central Power Controller */}
      <div className="power-container">
        <button 
          className={`power-button ${isOnline ? 'online' : 'offline'}`}
          onClick={() => onToggleSharing(!isOnline)}
          title={isOnline ? 'Stop Sharing Hotspot' : 'Start Sharing Hotspot'}
        >
          <Power />
        </button>
        <h2 className={`power-label ${isOnline ? 'online' : ''}`}>
          {isOnline ? 'Node Online' : 'Node Offline'}
        </h2>
        <p className="power-meta">
          {isOnline 
            ? 'Actively routing guest network traffic safely' 
            : 'Local hotspot disabled. Everyday providers are offline.'}
        </p>
      </div>

      {/* Speed Meters */}
      <div className="dashboard-grid">
        <div className="meter-card download">
          <div className="meter-title">Download Traffic</div>
          <div className="meter-value-wrap">
            <span className="meter-number">
              {isOnline ? telemetry.downloadSpeed.toFixed(1) : '0.0'}
            </span>
            <span className="meter-unit">Mbps</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px', color: 'var(--muted)' }}>
            <ArrowDown size={14} color="var(--cyan)" />
            <span>Incoming traffic routing</span>
          </div>
        </div>

        <div className="meter-card upload">
          <div className="meter-title">Upload Traffic</div>
          <div className="meter-value-wrap">
            <span className="meter-number">
              {isOnline ? telemetry.uploadSpeed.toFixed(1) : '0.0'}
            </span>
            <span className="meter-unit">Mbps</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '12px', color: 'var(--muted)' }}>
            <ArrowUp size={14} color="var(--blue)" />
            <span>Outgoing tunnel traffic</span>
          </div>
        </div>
      </div>

      {/* Stats Mini Row */}
      <div className="stats-row">
        <div className="stat-mini-card">
          <div className={`stat-mini-icon ${isOnline ? 'green' : ''}`}>
            <Users size={18} />
          </div>
          <div className="stat-mini-details">
            <span className="stat-mini-lbl">Active Clients</span>
            <span className="stat-mini-val">
              {isOnline ? `${telemetry.activeClientsCount} connected` : '0 connected'}
            </span>
          </div>
        </div>

        <div className="stat-mini-card">
          <div className={`stat-mini-icon ${isOnline ? 'cyan' : ''}`}>
            <HardDrive size={18} />
          </div>
          <div className="stat-mini-details">
            <span className="stat-mini-lbl">Total Data Shared</span>
            <span className="stat-mini-val">
              {telemetry.totalDataShared.toFixed(4)} GB
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
