import React, { useState } from 'react';
import { Wifi, ToggleLeft, ToggleRight, CheckCircle2, Copy, Check, XCircle } from 'lucide-react';

export default function Sharing({ 
  telemetry, 
  clients, 
  config, 
  onToggleSharing, 
  onDisconnectClient,
  uptimeString 
}) {
  const isOnline = telemetry.sharingEnabled;
  const [copied, setCopied] = useState(false);
  
  // Custom interactive speed limits states
  const [uploadLimit, setUploadLimit] = useState('50 Mbps');
  const [downloadLimit, setDownloadLimit] = useState('100 Mbps');

  const handleCopyIP = () => {
    navigator.clipboard.writeText("73.45.67.89");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
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
                ? 'Your device is actively sharing network resources with the Zoop Network.' 
                : 'Your device is offline and not routing any client traffic.'}
            </p>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <label className="toggle-switch">
            <input 
              type="checkbox" 
              checked={isOnline} 
              onChange={() => onToggleSharing(!isOnline)} 
            />
            <span className="toggle-slider"></span>
          </label>
          <button 
            type="button" 
            className={`btn ${isOnline ? 'btn-danger' : 'btn-primary'}`}
            onClick={() => onToggleSharing(!isOnline)}
          >
            {isOnline ? 'Pause Sharing' : 'Resume Sharing'}
          </button>
        </div>
      </div>

      {/* Sharing controls grid */}
      <div className="sharing-grid">
        {/* Speed limits card */}
        <div className="speed-limits-card">
          <h3>Limits & Network</h3>
          
          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Max Upload Speed</span>
              <span className="form-row-desc">Limit the maximum upload speed.</span>
            </div>
            <select 
              className="form-select-compact" 
              value={uploadLimit} 
              onChange={(e) => setUploadLimit(e.target.value)}
            >
              <option value="10 Mbps">10 Mbps</option>
              <option value="25 Mbps">25 Mbps</option>
              <option value="50 Mbps">50 Mbps</option>
              <option value="100 Mbps">100 Mbps</option>
              <option value="Unlimited">Unlimited</option>
            </select>
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Max Download Speed</span>
              <span className="form-row-desc">Limit the maximum download speed.</span>
            </div>
            <select 
              className="form-select-compact" 
              value={downloadLimit} 
              onChange={(e) => setDownloadLimit(e.target.value)}
            >
              <option value="50 Mbps">50 Mbps</option>
              <option value="100 Mbps">100 Mbps</option>
              <option value="200 Mbps">200 Mbps</option>
              <option value="500 Mbps">500 Mbps</option>
              <option value="Unlimited">Unlimited</option>
            </select>
          </div>

          <div className="form-row">
            <div className="form-row-left">
              <span className="form-row-label">Network</span>
              <span className="form-row-desc">Your device connection status.</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', fontWeight: '600', color: isOnline ? 'var(--green)' : 'var(--muted)' }}>
              {isOnline ? (
                <>
                  <CheckCircle2 size={15} />
                  <span>Connected</span>
                </>
              ) : (
                <>
                  <div className="status-dot red" style={{ width: '8px', height: '8px', display: 'inline-block' }}></div>
                  <span>Disconnected</span>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Mini status cards on the right */}
        <div className="sharing-mini-cards">
          <div className="sharing-mini-card">
            <span className="lbl">IP Address</span>
            <div className="val-wrap">
              <span className="val">73.45.67.89</span>
              <button 
                type="button" 
                className="copy-btn" 
                onClick={handleCopyIP}
                title="Copy IP Address"
              >
                {copied ? <Check size={16} color="var(--green)" /> : <Copy size={16} />}
              </button>
            </div>
          </div>

          <div className="sharing-mini-card">
            <span className="lbl">Uptime</span>
            <div className="val-wrap" style={{ marginTop: '2px' }}>
              <span className="val">{isOnline ? uptimeString : '0d 0h 0m'}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Guest Connected Clients Section */}
      <div className="clients-section">
        <h3 className="clients-section-header">Active Guest Devices</h3>
        
        {!isOnline ? (
          <div className="empty-state">
            <Wifi size={32} style={{ color: 'var(--muted-dark)' }} />
            <h3>Sharing is Offline</h3>
            <p>Enable node sharing to allow guest client devices to discover and connect.</p>
          </div>
        ) : clients.length === 0 ? (
          <div className="empty-state">
            <Wifi size={32} style={{ color: 'var(--muted-dark)' }} />
            <h3>No Clients Connected</h3>
            <p>Waiting for nearby guest devices to discover your sharing daemon...</p>
          </div>
        ) : (
          <div className="table-wrap">
            <table className="ztable">
              <thead>
                <tr>
                  <th>Client Device</th>
                  <th>IP Address</th>
                  <th>Connection Time</th>
                  <th>Current Speed</th>
                  <th>Data Shared</th>
                  <th style={{ textAlign: 'right' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {clients.map((client) => (
                  <tr key={client.mac}>
                    <td>
                      <div>
                        <span className="client-name">{client.name}</span>
                        <span className="client-mac">{client.mac}</span>
                      </div>
                    </td>
                    <td>
                      <span className="client-ip">{client.ip}</span>
                    </td>
                    <td>
                      <span className="client-time">{client.connectedAt}</span>
                    </td>
                    <td>
                      <span className="client-speed">{client.speed.toFixed(1)} Mbps</span>
                    </td>
                    <td>
                      <span className="client-data">{client.dataShared.toFixed(1)} MB</span>
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <button 
                        type="button"
                        className="btn btn-danger btn-sm"
                        onClick={() => onDisconnectClient(client.mac)}
                        title="Disconnect Client"
                        style={{ padding: '4px 10px', fontSize: '11.5px', height: '28px' }}
                      >
                        <XCircle size={13} />
                        <span>Kick</span>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
