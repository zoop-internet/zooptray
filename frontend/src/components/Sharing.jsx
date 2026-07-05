import React, { useState } from 'react';
import { Wifi, Copy, Check, Zap, Shield, Key, Loader } from 'lucide-react';
import { toast } from './Toast';
import { CreateConnectionProfile } from '../../wailsjs/go/main/NodeService';

export default function Sharing({ 
  telemetry, 
  config, 
  onToggleSharing, 
  uptimeString 
}) {
  const isOnline = telemetry.tunnelActive;
  const publicIP = telemetry.publicIP || 'Not detected';
  const [copied, setCopied] = useState(false);
  const [profileLoading, setProfileLoading] = useState(false);

  const handleCopyIP = () => {
    if (!telemetry.publicIP) return;
    navigator.clipboard.writeText(telemetry.publicIP);
    setCopied(true);
    toast.success('IP address copied');
    setTimeout(() => setCopied(false), 2000);
  };

  const handleToggle = () => {
    onToggleSharing(!isOnline);
    toast(isOnline ? 'Sharing paused' : 'Sharing enabled', isOnline ? 'warning' : 'success');
  };

  const handleGenerateProfile = async () => {
    if (!config?.node?.id) {
      toast.error('Node ID not available');
      return;
    }
    setProfileLoading(true);
    try {
      const profile = await CreateConnectionProfile(config.node.id);
      navigator.clipboard.writeText(JSON.stringify(profile, null, 2));
      toast.success('Config generated and copied to clipboard');
    } catch (err) {
      toast.error(err?.message || 'Failed to generate config');
    } finally {
      setProfileLoading(false);
    }
  };

  return (
    <div className="sharing-layout">
      {/* Top Status Card */}
      <div className="sharing-status-card">
        <div className="sharing-status-info">
          <div className={`sharing-status-icon ${isOnline ? '' : 'offline'}`}>
            <Wifi size={22} />
          </div>
          <div className="sharing-status-details">
            <h3>{isOnline ? 'Sharing is enabled' : 'Sharing is disabled'}</h3>
            <p>
              {isOnline
                ? 'Your device is actively routing guest network traffic.'
                : 'Your device is offline and not routing any traffic.'}
            </p>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <label className="toggle-switch">
            <input
              type="checkbox"
              checked={isOnline}
              onChange={handleToggle}
            />
            <span className="toggle-slider"></span>
          </label>
          <button
            type="button"
            className={`btn ${isOnline ? 'btn-danger' : 'btn-primary'}`}
            onClick={handleToggle}
          >
            {isOnline ? 'Pause Sharing' : 'Start Sharing'}
          </button>
          <button
            type="button"
            className="btn"
            onClick={handleGenerateProfile}
            disabled={profileLoading}
          >
            {profileLoading ? <Loader size={14} className="wg-spin" /> : <Key size={14} />}
            Client Config
          </button>
        </div>
      </div>

      {/* Controls + Stats grid */}
      <div className="sharing-grid">
        <div className="speed-limits-card">
          <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Zap size={16} color="var(--blue)" />
            Live Throughput
          </h3>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Upload Speed</span>
              <span className="form-row-desc">Measured from WireGuard transfer counters.</span>
            </div>
            <div style={{ fontSize: '13.5px', fontFamily: 'var(--mono)', color: 'var(--muted)', background: 'var(--surface-soft)', padding: '6px 12px', borderRadius: '6px', border: '1px solid var(--line)' }}>
              {isOnline ? `${telemetry.uploadSpeed.toFixed(1)} Mbps` : '0.0 Mbps'}
            </div>
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Download Speed</span>
              <span className="form-row-desc">Measured from WireGuard transfer counters.</span>
            </div>
            <div style={{ fontSize: '13.5px', fontFamily: 'var(--mono)', color: 'var(--muted)', background: 'var(--surface-soft)', padding: '6px 12px', borderRadius: '6px', border: '1px solid var(--line)' }}>
              {isOnline ? `${telemetry.downloadSpeed.toFixed(1)} Mbps` : '0.0 Mbps'}
            </div>
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Network Status</span>
              <span className="form-row-desc">Live connection state of your node.</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '7px', fontWeight: '600', fontSize: '14px', color: isOnline ? 'var(--green)' : 'var(--muted)' }}>
              <div className={`status-dot ${isOnline ? 'green' : 'red'}`}></div>
              {isOnline ? 'Connected' : 'Disconnected'}
            </div>
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Routing Policy</span>
              <span className="form-row-desc">Traffic balancing strategy.</span>
            </div>
            <div style={{ fontSize: '13.5px', fontFamily: 'var(--mono)', color: 'var(--muted)', background: 'var(--surface-soft)', padding: '6px 12px', borderRadius: '6px', border: '1px solid var(--line)' }}>
              {config?.sharing?.routingPolicy || 'balanced'}
            </div>
          </div>
        </div>

        {/* Mini info cards */}
        <div className="sharing-mini-cards">
          <div className="sharing-mini-card">
            <span className="lbl">IP Address</span>
            <div className="val-wrap">
              <span className="val">{publicIP}</span>
              <button type="button" className="copy-btn" onClick={handleCopyIP} title="Copy IP" disabled={!telemetry.publicIP}>
                {copied ? <Check size={16} color="var(--green)" /> : <Copy size={16} />}
              </button>
            </div>
          </div>

          <div className="sharing-mini-card">
            <span className="lbl">Uptime</span>
            <div className="val-wrap">
              <span className="val">{isOnline ? uptimeString : '—'}</span>
            </div>
          </div>

          <div className="sharing-mini-card">
            <span className="lbl">Active Clients</span>
            <div className="val-wrap">
              <span className="val" style={{ color: isOnline && telemetry.activeClientsCount > 0 ? 'var(--green)' : 'var(--ink)' }}>
                {isOnline ? telemetry.activeClientsCount : 0}
              </span>
              <span style={{ fontSize: '12px', color: 'var(--muted)' }}>/ {config?.sharing?.maxClients || 10} max</span>
            </div>
          </div>
        </div>
      </div>

      {/* Security note */}
      <div className="sharing-security-note">
        <Shield size={15} />
        <span>Guest traffic uses the local WireGuard tunnel when sharing is active. LAN isolation depends on the generated firewall rules.</span>
      </div>
    </div>
  );
}
