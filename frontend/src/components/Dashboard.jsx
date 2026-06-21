import React, { useMemo, useState } from 'react';
import {
  Check, Pause, Play, ArrowUp, CheckCircle, AlertCircle, Copy,
  Shield, Key, Globe, Loader, AlertTriangle, ChevronDown, ChevronUp, Server
} from 'lucide-react';
import { toast } from './Toast';

export default function Dashboard({ 
  telemetry, 
  config, 
  onToggleSharing, 
  bandwidthHistory, 
  dataHistory,
  uptimeString,
  wgInfo,
  wgLoading,
}) {
  const isOnline = telemetry.sharingEnabled;
  const [showWgModal, setShowWgModal] = useState(false);

  // Automatically reopen modal when loading initiates
  React.useEffect(() => {
    if (wgLoading) {
      setShowWgModal(true);
    }
  }, [wgLoading]);

  const handleCopyNodeId = () => {
    const id = config?.node?.id || 'node-alpha-7';
    navigator.clipboard.writeText(id);
    setCopiedId(true);
    toast.info(`Node ID copied: ${id}`);
    setTimeout(() => setCopiedId(false), 2000);
  };

  const handleCopyField = (key, value) => {
    navigator.clipboard.writeText(value);
    setCopiedField(key);
    toast.success(`${key} copied!`);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleToggle = () => {
    const next = !isOnline;
    onToggleSharing(next);
    if (next) {
      setShowWgModal(true);
      toast.success('Node started — configuring Zoop tunnel…');
    } else {
      toast.warning('Node paused — sharing stopped');
    }
  };

  // Render a responsive SVG sparkline path
  const makeSparklinePath = (history, width, height) => {
    if (!history || history.length === 0) return "";
    const minVal = Math.min(...history);
    const maxVal = Math.max(...history);
    const range = maxVal - minVal || 1;
    const points = history.map((val, idx) => {
      const x = (idx / (history.length - 1)) * width;
      const y = height - ((val - minVal) / range) * (height - 8) - 4;
      return `${x},${y}`;
    });
    return `M ${points.join(" L ")}`;
  };

  const bandwidthPath = useMemo(() => makeSparklinePath(bandwidthHistory, 220, 50), [bandwidthHistory]);
  const dataPath      = useMemo(() => makeSparklinePath(dataHistory, 220, 50), [dataHistory]);

  const currentTotalSpeed = isOnline ? (telemetry.downloadSpeed + telemetry.uploadSpeed) : 0;

  // ── Zoop Config fields to display ─────────────────────────────────────
  const wgFields = wgInfo ? [
    { key: 'Public Key',   value: wgInfo.publicKey,            icon: <Key size={14} />,    mono: true  },
    { key: 'Private Key',  value: wgInfo.privateKey,           icon: <Shield size={14} />, mono: true, sensitive: true },
    { key: 'Listen Port',  value: String(wgInfo.listenPort),   icon: <Globe size={14} />,  mono: false },
    { key: 'Address',      value: wgInfo.address,              icon: <Server size={14} />, mono: false },
    { key: 'Interface',    value: wgInfo.wgInterface,          icon: <Globe size={14} />,  mono: false },
    { key: 'Config Path',  value: wgInfo.configPath,           icon: <Key size={14} />,    mono: true  },
  ] : [];

  // Client-side peer block template
  const peerBlock = wgInfo
    ? `[Interface]\nPrivateKey = <CLIENT_PRIVATE_KEY>\nAddress = 10.200.0.2/32\nDNS = 1.1.1.1\n\n[Peer]\nPublicKey = ${wgInfo.publicKey}\nEndpoint = <YOUR_PUBLIC_IP>:${wgInfo.listenPort}\nAllowedIPs = 0.0.0.0/0, ::/0\nPersistentKeepalive = 25`
    : '';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

      {/* ── Zoop Connection Modal Popup Overlay ───────────────────────────────── */}
      {showWgModal && (wgLoading || wgInfo) && (
        <div className="wg-modal-overlay" onClick={() => !wgLoading && setShowWgModal(false)}>
          <div className="wg-modal-content" onClick={(e) => e.stopPropagation()}>
            
            {/* Loading Modal State */}
            {wgLoading && (
              <div className="wg-loading-banner">
                <div className="wg-loading-spinner">
                  <Loader size={24} className="wg-spin" />
                </div>
                <div>
                  <div className="wg-loading-title">Configuring Zoop Network…</div>
                  <div className="wg-loading-sub">Generating keys, writing config, bringing up interface.</div>
                </div>
              </div>
            )}

            {/* Loaded Config Profile State */}
            {wgInfo && !wgLoading && (
              <div className={`wg-config-card`}>
                
                {/* Modal Header */}
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--line)', paddingBottom: '14px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div className={`wg-status-dot ${wgInfo.active ? 'on' : 'off'}`} />
                    <span style={{ fontSize: '16px', fontWeight: '600', color: 'var(--ink)' }}>
                      Zoop Connection Profile
                    </span>
                    {wgInfo.active
                      ? <span className="wg-badge active">Active</span>
                      : <span className="wg-badge error"><AlertTriangle size={11} /> Setup needed</span>
                    }
                  </div>
                </div>

                {wgInfo.error && (
                  <div className="wg-error-notice" style={{ borderRadius: '8px', border: '1px solid rgba(255, 205, 26, 0.2)' }}>
                    <AlertTriangle size={13} style={{ flexShrink: 0, marginTop: '2px' }} />
                    <span>{wgInfo.error}</span>
                  </div>
                )}

                <div className="wg-config-body" style={{ padding: '4px 0 0' }}>
                  {/* Server fields */}
                  <div className="wg-section-label">Server Configuration</div>
                  <div className="wg-fields">
                    {wgFields.map(f => (
                      <div className="wg-field-row" key={f.key}>
                        <div className="wg-field-icon">{f.icon}</div>
                        <div className="wg-field-label">{f.key}</div>
                        <div className={`wg-field-value ${f.mono ? 'mono' : ''}`}>
                          {f.sensitive
                            ? <span className="wg-sensitive">••••••••••••••••••••••••••••••••••••••••••</span>
                            : <span>{f.value}</span>
                          }
                        </div>
                        <button
                          type="button"
                          className="wg-copy-btn"
                          title={`Copy ${f.key}`}
                          onClick={() => handleCopyField(f.key, f.value)}
                        >
                          {copiedField === f.key
                            ? <Check size={13} color="var(--green)" />
                            : <Copy size={13} />
                          }
                        </button>
                      </div>
                    ))}
                  </div>

                  {/* Client peer block */}
                  <div className="wg-section-label" style={{ marginTop: '20px' }}>
                    Client Configuration Block
                    <span className="wg-section-hint">— paste this into each connecting device's configuration</span>
                  </div>
                  <div className="wg-peer-block">
                    <pre>{peerBlock}</pre>
                    <button
                      type="button"
                      className="wg-copy-btn wg-copy-block"
                      title="Copy peer block"
                      onClick={() => handleCopyField('Peer Block', peerBlock)}
                    >
                      {copiedField === 'Peer Block'
                        ? <><Check size={13} color="var(--green)" /> Copied!</>
                        : <><Copy size={13} /> Copy all</>
                      }
                    </button>
                  </div>
                </div>

                {/* Close Button at bottom of modal */}
                <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '10px' }}>
                  <button 
                    type="button" 
                    className="btn btn-secondary" 
                    onClick={() => setShowWgModal(false)}
                    style={{ minWidth: '100px' }}
                  >
                    Done
                  </button>
                </div>

              </div>
            )}

          </div>
        </div>
      )}

      {/* ── Node Status Card ──────────────────────────────────────────────────── */}
      <div className="node-status-card">
        <div className="node-status-left">
          <div className="status-ring-container">
            <svg className="status-ring-circle" viewBox="0 0 100 100">
              <circle className="status-ring-bg" cx="50" cy="50" r="40" />
              <circle 
                className={`status-ring-active ${isOnline ? '' : 'offline'}`}
                cx="50" cy="50" r="40" 
                strokeDashoffset={isOnline ? 0 : 251.2}
              />
            </svg>
            <div className={`status-ring-center ${isOnline ? '' : 'offline'}`}>
              {wgLoading
                ? <Loader size={18} className="wg-spin" />
                : isOnline ? <Check size={20} strokeWidth={3} /> : <Pause size={18} strokeWidth={3} />
              }
            </div>
          </div>

          <div className="node-status-info">
            <div className="node-status-msg">
              <h2>
                {wgLoading
                  ? 'Establishing Zoop Connection…'
                  : isOnline ? 'Your node is connected' : 'Your node is disconnected'
                }
              </h2>
              <p>
                {wgLoading
                  ? 'Configuring keys and interface. This takes a few seconds.'
                  : isOnline 
                    ? 'Your device is actively sharing network resources with the decentralized network.' 
                    : 'Hotspot disabled. everyday providers are offline.'
                }
              </p>
            </div>
            <button 
              type="button" 
              className="btn btn-primary"
              onClick={handleToggle}
              disabled={wgLoading}
              style={{ alignSelf: 'flex-start', minWidth: '140px', opacity: wgLoading ? 0.6 : 1 }}
            >
              {isOnline ? (
                <>
                  <Pause size={14} />
                  <span>Pause Sharing</span>
                </>
              ) : (
                <>
                  <Play size={14} />
                  <span>{wgLoading ? 'Starting…' : 'Start Sharing'}</span>
                </>
              )}
            </button>
          </div>
        </div>

        <div className="node-status-right">
          <div className="meta-row">
            <span className="meta-label">Node ID</span>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span className="meta-value">{config ? config.node.id : '…'}</span>
              <button type="button" className="copy-btn" onClick={handleCopyNodeId} title="Copy Node ID">
                {copiedId ? <Check size={14} color="var(--green)" /> : <Copy size={14} />}
              </button>
            </div>
          </div>
          <div className="meta-row">
            <span className="meta-label">Uptime</span>
            <span className="meta-value">{isOnline ? uptimeString : '0d 0h 0m'}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">Peers Connected</span>
            <span className="meta-value">{isOnline ? telemetry.activeClientsCount : 0}</span>
          </div>
          <div className="meta-row">
            <span className="meta-label">IP Address</span>
            <span className="meta-value">73.45.67.89</span>
          </div>
        </div>
      </div>

      {/* ── Telemetry Widgets Grid ────────────────────────────────────────────── */}
      <div className="telemetry-grid">
        {/* Bandwidth Shared Card */}
        <div className="telemetry-card bandwidth">
          <div className="telemetry-header">
            <span className="telemetry-label">Bandwidth Shared</span>
            <div className="telemetry-value-wrap">
              <span className="telemetry-value">{currentTotalSpeed.toFixed(1)}</span>
              <span className="telemetry-unit">Mbps</span>
            </div>
          </div>
          
          <div className="sparkline-container">
            {isOnline && bandwidthHistory.length > 1 ? (
              <svg className="sparkline-svg" width="220" height="50">
                <path className="sparkline-path bandwidth" d={bandwidthPath} />
              </svg>
            ) : (
              <div style={{ fontSize: '11px', color: 'var(--muted-dark)', paddingBottom: '6px' }}>
                {isOnline ? 'Collecting stats...' : 'Daemon offline'}
              </div>
            )}
          </div>

          <div className="telemetry-footer up">
            <ArrowUp size={12} />
            <span>8.7 Mbps vs yesterday</span>
          </div>
        </div>

        {/* Data Shared Today Card */}
        <div className="telemetry-card data">
          <div className="telemetry-header">
            <span className="telemetry-label">Data Shared Today</span>
            <div className="telemetry-value-wrap">
              <span className="telemetry-value">{telemetry.totalDataShared.toFixed(2)}</span>
              <span className="telemetry-unit">GB</span>
            </div>
          </div>
          
          <div className="sparkline-container">
            {isOnline && dataHistory.length > 1 ? (
              <svg className="sparkline-svg" width="220" height="50">
                <path className="sparkline-path data" d={dataPath} />
              </svg>
            ) : (
              <div style={{ fontSize: '11px', color: 'var(--muted-dark)', paddingBottom: '6px' }}>
                {isOnline ? 'Collecting stats...' : 'Daemon offline'}
              </div>
            )}
          </div>

          <div className="telemetry-footer up" style={{ color: 'var(--green)' }}>
            <ArrowUp size={12} />
            <span>15.8% vs yesterday</span>
          </div>
        </div>

        {/* Status Card */}
        <div className="telemetry-card health">
          <div className="telemetry-header">
            <span className="telemetry-label">Status</span>
            <div className="telemetry-value-wrap" style={{ marginTop: '4px' }}>
              <span className="telemetry-value" style={{ fontSize: '22px', fontFamily: 'var(--font)', fontWeight: '700' }}>
                {wgLoading ? 'Initializing' : isOnline ? 'Healthy' : 'Offline'}
              </span>
            </div>
          </div>
          
          <div style={{ fontSize: '12px', color: 'var(--muted)', marginTop: '8px' }}>
            {wgLoading
              ? 'Zoop connection starting up…'
              : isOnline ? 'All systems operational' : 'Sharing is currently disabled by policy'
            }
          </div>

          <div className="status-badge-container">
            <div className={`status-badge-circle ${isOnline ? '' : 'offline'}`}>
              {wgLoading
                ? <Loader size={18} className="wg-spin" />
                : isOnline ? <CheckCircle size={18} /> : <AlertCircle size={18} />
              }
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
