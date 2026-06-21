import React, { useState } from 'react';
import { Wifi, CheckCircle2, Copy, Check, Zap, Shield } from 'lucide-react';
import Dropdown from './Dropdown';
import { toast } from './Toast';

export default function Sharing({ 
  telemetry, 
  config, 
  onToggleSharing, 
  uptimeString 
}) {
  const isOnline = telemetry.sharingEnabled;
  const [copied, setCopied] = useState(false);

  const [uploadLimit, setUploadLimit] = useState('50mbps');
  const [downloadLimit, setDownloadLimit] = useState('100mbps');

  const uploadOptions = [
    { label: '10 Mbps',   value: '10mbps' },
    { label: '25 Mbps',   value: '25mbps' },
    { label: '50 Mbps',   value: '50mbps' },
    { label: '100 Mbps',  value: '100mbps' },
    { label: 'Unlimited', value: 'unlimited' },
  ];
  const downloadOptions = [
    { label: '50 Mbps',   value: '50mbps' },
    { label: '100 Mbps',  value: '100mbps' },
    { label: '200 Mbps',  value: '200mbps' },
    { label: '500 Mbps',  value: '500mbps' },
    { label: 'Unlimited', value: 'unlimited' },
  ];

  const handleCopyIP = () => {
    navigator.clipboard.writeText('73.45.67.89');
    setCopied(true);
    toast.success('IP address copied to clipboard');
    setTimeout(() => setCopied(false), 2000);
  };

  const handleToggle = () => {
    const next = !isOnline;
    onToggleSharing(next);
    if (next) {
      toast.success('Sharing enabled — hotspot is now visible');
    } else {
      toast.warning('Sharing paused — hotspot is offline');
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
        </div>
      </div>

      {/* Controls + Stats grid */}
      <div className="sharing-grid">
        {/* Speed limits card */}
        <div className="speed-limits-card">
          <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Zap size={16} color="var(--blue)" />
            Speed Limits
          </h3>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Max Upload Speed</span>
              <span className="form-row-desc">Cap outbound bandwidth per client.</span>
            </div>
            <Dropdown value={uploadLimit} onChange={setUploadLimit} options={uploadOptions} />
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Max Download Speed</span>
              <span className="form-row-desc">Cap inbound bandwidth per client.</span>
            </div>
            <Dropdown value={downloadLimit} onChange={setDownloadLimit} options={downloadOptions} />
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
              <span className="val">73.45.67.89</span>
              <button type="button" className="copy-btn" onClick={handleCopyIP} title="Copy IP">
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
        <span>All guest traffic is routed through the Zoop encrypted relay. Client devices cannot access your local network.</span>
      </div>
    </div>
  );
}
