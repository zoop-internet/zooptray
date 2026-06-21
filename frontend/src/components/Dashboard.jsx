import React, { useMemo } from 'react';
import { Check, Pause, Play, ArrowUp, CheckCircle, AlertCircle } from 'lucide-react';

export default function Dashboard({ 
  telemetry, 
  config, 
  onToggleSharing, 
  bandwidthHistory, 
  dataHistory,
  uptimeString 
}) {
  const isOnline = telemetry.sharingEnabled;

  // Render a responsive SVG sparkline path
  const makeSparklinePath = (history, width, height) => {
    if (!history || history.length === 0) return "";
    const minVal = Math.min(...history);
    const maxVal = Math.max(...history);
    const range = maxVal - minVal || 1;
    
    // Add small pad to top and bottom of chart
    const points = history.map((val, idx) => {
      const x = (idx / (history.length - 1)) * width;
      const y = height - ((val - minVal) / range) * (height - 8) - 4;
      return `${x},${y}`;
    });
    
    return `M ${points.join(" L ")}`;
  };

  const bandwidthPath = useMemo(() => makeSparklinePath(bandwidthHistory, 220, 50), [bandwidthHistory]);
  const dataPath = useMemo(() => makeSparklinePath(dataHistory, 220, 50), [dataHistory]);

  const currentTotalSpeed = isOnline ? (telemetry.downloadSpeed + telemetry.uploadSpeed) : 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      {/* Node Status Card */}
      <div className="node-status-card">
        <div className="node-status-left">
          <div className="status-ring-container">
            <svg className="status-ring-circle" viewBox="0 0 100 100">
              <circle className="status-ring-bg" cx="50" cy="50" r="40" />
              <circle 
                className={`status-ring-active ${isOnline ? '' : 'offline'}`}
                cx="50" 
                cy="50" 
                r="40" 
                strokeDashoffset={isOnline ? 0 : 251.2}
              />
            </svg>
            <div className={`status-ring-center ${isOnline ? '' : 'offline'}`}>
              {isOnline ? <Check size={20} strokeWidth={3} /> : <Pause size={18} strokeWidth={3} />}
            </div>
          </div>

          <div className="node-status-info">
            <div className="node-status-msg">
              <h2>{isOnline ? 'Your node is connected' : 'Your node is disconnected'}</h2>
              <p>
                {isOnline 
                  ? 'Your device is actively sharing network resources with the decentralized network.' 
                  : 'Hotspot disabled. everyday providers are offline.'}
              </p>
            </div>
            <button 
              type="button" 
              className="btn btn-primary"
              onClick={() => onToggleSharing(!isOnline)}
              style={{ alignSelf: 'flex-start', minWidth: '130px' }}
            >
              {isOnline ? (
                <>
                  <Pause size={14} />
                  <span>Pause Sharing</span>
                </>
              ) : (
                <>
                  <Play size={14} />
                  <span>Start Sharing</span>
                </>
              )}
            </button>
          </div>
        </div>

        <div className="node-status-right">
          <div className="meta-row">
            <span className="meta-label">Node ID</span>
            <span className="meta-value">{config ? config.node.id : 'loading...'}</span>
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

      {/* Telemetry Widgets Grid */}
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
                <path 
                  className="sparkline-path bandwidth" 
                  d={bandwidthPath} 
                />
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
                <path 
                  className="sparkline-path data" 
                  d={dataPath} 
                />
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
                {isOnline ? 'Healthy' : 'Offline'}
              </span>
            </div>
          </div>
          
          <div style={{ fontSize: '12px', color: 'var(--muted)', marginTop: '8px' }}>
            {isOnline ? 'All systems operational' : 'Sharing is currently disabled by policy'}
          </div>

          <div className="status-badge-container">
            <div className={`status-badge-circle ${isOnline ? '' : 'offline'}`}>
              {isOnline ? <CheckCircle size={18} /> : <AlertCircle size={18} />}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
