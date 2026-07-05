import React, { useMemo, useState, useEffect } from 'react';
import {
  Check, Pause, Play, Copy, Key, Globe, Server,
  Loader, TrendingUp, TrendingDown, HardDrive, Users,
  Activity, Download, Upload, ExternalLink, Eye
} from 'lucide-react';
import { toast } from './Toast';
import { CreateConnectionProfile } from '../../wailsjs/go/main/NodeService';

const SPEED_WIDTH = 180;
const SPEED_HEIGHT = 40;

function MiniSparkline({ data, color }) {
  if (!data || data.length < 2) return <div className="mini-sparkline-placeholder">—</div>;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * SPEED_WIDTH;
    const y = SPEED_HEIGHT - ((v - min) / range) * (SPEED_HEIGHT - 6) - 3;
    return `${x},${y}`;
  });
  const path = `M ${points.join(' L ')}`;
  return (
    <svg width={SPEED_WIDTH} height={SPEED_HEIGHT} className="mini-sparkline">
      <path d={path} fill="none" stroke={color} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export default function Dashboard({
  telemetry, config, onToggleSharing,
  bandwidthHistory, dataHistory, uptimeString,
  wgInfo, wgLoading, clients,
}) {
  const isOnline = telemetry.tunnelActive;
  const [showWgModal, setShowWgModal] = useState(false);
  const [copiedId, setCopiedId] = useState(false);
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileData, setProfileData] = useState(null);

  useEffect(() => {
    if (wgLoading) setShowWgModal(true);
  }, [wgLoading]);

  const currentTotal = telemetry.uploadSpeed + telemetry.downloadSpeed;

  const handleCopyNodeId = () => {
    const id = config?.node?.id || 'node-alpha-7';
    navigator.clipboard.writeText(id);
    setCopiedId(true);
    toast.info('Node ID copied');
    setTimeout(() => setCopiedId(false), 2000);
  };

  const handleToggle = () => {
    onToggleSharing(!isOnline);
    if (!isOnline) {
      toast.success('Starting Zoop tunnel…');
    } else {
      toast.warning('Sharing stopped');
    }
  };

  const handleGenerateProfile = async () => {
    if (!config?.node?.id) {
      toast.error('Node ID not available');
      return;
    }
    setProfileLoading(true);
    try {
      const profile = await CreateConnectionProfile(config.node.id);
      setProfileData(profile);
      setShowWgModal(true);
      toast.success('Connection profile generated');
    } catch (err) {
      toast.error(err?.message || 'Failed to generate profile');
    } finally {
      setProfileLoading(false);
    }
  };

  const stats = useMemo(() => [
    {
      id: 'upload',
      label: 'Upload',
      value: telemetry.uploadSpeed.toFixed(1),
      unit: 'Mbps',
      icon: <Upload size={18} />,
      color: 'var(--cyan)',
      barColor: '#08f2ff',
    },
    {
      id: 'download',
      label: 'Download',
      value: telemetry.downloadSpeed.toFixed(1),
      unit: 'Mbps',
      icon: <Download size={18} />,
      color: 'var(--green)',
      barColor: '#14f06d',
    },
    {
      id: 'data',
      label: 'Data Today',
      value: telemetry.totalDataShared.toFixed(2),
      unit: 'GB',
      icon: <HardDrive size={18} />,
      color: 'var(--blue)',
      barColor: '#126cff',
    },
    {
      id: 'clients',
      label: 'Active Clients',
      value: String(telemetry.activeClientsCount),
      unit: 'devices',
      icon: <Users size={18} />,
      color: 'var(--purple)',
      barColor: '#8a3ffc',
    },
  ], [telemetry]);

  const upSpeed = isOnline ? telemetry.uploadSpeed : 0;
  const downSpeed = isOnline ? telemetry.downloadSpeed : 0;
  const maxSpeed = Math.max(upSpeed, downSpeed, 1);

  const recentClients = (clients || []).slice(0, 4);

  return (
    <div className="dash">

      {/* ── Connection Modal ── */}
      {showWgModal && (wgLoading || wgInfo || profileData) && (
        <div className="wg-modal-overlay" onClick={() => { if (!wgLoading) { setShowWgModal(false); setProfileData(null); } }}>
          <div className="wg-modal-content" onClick={e => e.stopPropagation()}>
            {wgLoading && (
              <div className="wg-loading-banner">
                <div className="wg-loading-spinner"><Loader size={24} className="wg-spin" /></div>
                <div>
                  <div className="wg-loading-title">Zooping you in…</div>
                  <div className="wg-loading-sub">Generating credentials and establishing tunnel.</div>
                </div>
              </div>
            )}

            {(wgInfo || profileData) && !wgLoading && (
              <div className="wg-config-card">
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--line)', paddingBottom: '14px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div className={`wg-status-dot ${wgInfo?.active ? 'on' : 'off'}`} />
                    <span style={{ fontSize: '16px', fontWeight: '600' }}>WireGuard Connection Profile</span>
                  </div>
                </div>
                <div className="wg-config-body" style={{ padding: '16px 0 0' }}>
                  <div className="wg-fields">
                    {[
                      { label: 'Public Key', value: wgInfo?.publicKey || profileData?.publicKey || '—', icon: <Key size={14} /> },
                      { label: 'Listen Port', value: String(wgInfo?.listenPort || profileData?.listenPort || '51820'), icon: <Globe size={14} /> },
                      { label: 'Address', value: wgInfo?.address || profileData?.address || '—', icon: <Server size={14} /> },
                      { label: 'Interface', value: wgInfo?.wgInterface || profileData?.interface || '—', icon: <Activity size={14} /> },
                    ].map(f => (
                      <div className="wg-field-row" key={f.label}>
                        <div className="wg-field-icon">{f.icon}</div>
                        <div className="wg-field-label">{f.label}</div>
                        <div className="wg-field-value mono">{f.value}</div>
                      </div>
                    ))}
                  </div>
                </div>
                <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: '16px' }}>
                  <button type="button" className="btn" onClick={() => { setShowWgModal(false); setProfileData(null); }} style={{ minWidth: '100px' }}>Done</button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ── Hero Status Banner ── */}
      <div className={`dash-hero ${isOnline ? 'online' : ''}`}>
        <div className="dash-hero-left">
          <div className={`dash-hero-indicator ${isOnline ? 'online' : ''}`}>
            {wgLoading ? <Loader size={22} className="wg-spin" /> : isOnline ? <Check size={22} /> : <Pause size={22} />}
          </div>
          <div>
            <div className="dash-hero-title">
              {wgLoading ? 'Starting…' : isOnline ? 'Connected' : 'Offline'}
            </div>
            <div className="dash-hero-sub">
              {wgLoading ? 'Establishing tunnel' : isOnline ? uptimeString : 'Node is idle'}
            </div>
          </div>
        </div>
        <div className="dash-hero-right">
          <div className="dash-hero-metrics">
            <div className="dash-hero-metric">
              <span className="dash-hero-metric-label">Node ID</span>
              <span className="dash-hero-metric-value mono">{config?.node?.id || '…'}</span>
            </div>
            <div className="dash-hero-metric">
              <span className="dash-hero-metric-label">IP</span>
              <span className="dash-hero-metric-value">{telemetry.publicIP || '—'}</span>
            </div>
            <div className="dash-hero-metric">
              <span className="dash-hero-metric-label">Peers</span>
              <span className="dash-hero-metric-value">{telemetry.activeClientsCount}</span>
            </div>
          </div>
        </div>
        {isOnline && (
          <div className="dash-hero-accent" style={{ background: `linear-gradient(90deg, #14f06d, #08f2ff)` }} />
        )}
      </div>

      {/* ── Quick Actions Row ── */}
      <div className="dash-actions">
        <button type="button" className="btn btn-primary dash-action-btn" onClick={handleToggle} disabled={wgLoading}>
          {isOnline ? <><Pause size={15} /> Stop Sharing</> : <><Play size={15} /> Start Sharing</>}
        </button>
        <button type="button" className="btn dash-action-btn" onClick={handleCopyNodeId}>
          <Copy size={15} /> {copiedId ? 'Copied!' : 'Copy Node ID'}
        </button>
        <button type="button" className="btn dash-action-btn" onClick={handleGenerateProfile} disabled={profileLoading}>
          {profileLoading ? <Loader size={15} className="wg-spin" /> : <Key size={15} />}
          Generate Config
        </button>
        <button type="button" className="btn dash-action-btn" onClick={() => window.open('https://console.zoopnetwork.com', '_blank')}>
          <ExternalLink size={15} /> Dashboard
        </button>
      </div>

      {/* ── Bandwidth Meter ── */}
      <div className="dash-bw-card">
        <div className="dash-bw-header">
          <span className="dash-bw-title">Live Throughput</span>
          <span className="dash-bw-total">{currentTotal.toFixed(1)} <small>Mbps</small></span>
        </div>
        <div className="dash-bw-bars">
          <div className="dash-bw-bar-wrap">
            <span className="dash-bw-bar-label"><Upload size={12} /> Up</span>
            <div className="dash-bw-bar-track">
              <div className="dash-bw-bar-fill upload" style={{ width: `${(upSpeed / maxSpeed) * 100}%` }} />
            </div>
            <span className="dash-bw-bar-val">{upSpeed.toFixed(1)}</span>
          </div>
          <div className="dash-bw-bar-wrap">
            <span className="dash-bw-bar-label"><Download size={12} /> Down</span>
            <div className="dash-bw-bar-track">
              <div className="dash-bw-bar-fill download" style={{ width: `${(downSpeed / maxSpeed) * 100}%` }} />
            </div>
            <span className="dash-bw-bar-val">{downSpeed.toFixed(1)}</span>
          </div>
        </div>
        <div className="dash-bw-sparklines">
          <div className="dash-bw-sparkline-group">
            <span className="dash-bw-sparkline-label" style={{ color: 'var(--cyan)' }}>Upload</span>
            <MiniSparkline data={bandwidthHistory} color="#08f2ff" />
          </div>
          <div className="dash-bw-sparkline-group">
            <span className="dash-bw-sparkline-label" style={{ color: 'var(--green)' }}>Download</span>
            <MiniSparkline data={dataHistory} color="#14f06d" />
          </div>
        </div>
      </div>

      {/* ── Stat Tiles ── */}
      <div className="dash-stats">
        {stats.map(s => (
          <div key={s.id} className="dash-stat-tile" style={{ borderTopColor: s.color }}>
            <div className="dash-stat-tile-header">
              <span style={{ color: s.color }}>{s.icon}</span>
              <span className="dash-stat-tile-label">{s.label}</span>
            </div>
            <div className="dash-stat-tile-value">
              {s.value} <small>{s.unit}</small>
            </div>
            <div className="dash-stat-tile-bar">
              <div className="dash-stat-tile-bar-fill" style={{
                width: s.id === 'clients' ? `${Math.min(parseInt(s.value) * 10, 100)}%` : '100%',
                background: s.barColor,
              }} />
            </div>
          </div>
        ))}
      </div>

      {/* ── Recent Clients ── */}
      <div className="dash-clients-card">
        <div className="dash-clients-header">
          <span className="dash-clients-title"><Users size={15} /> Connected Devices</span>
          <span className="dash-clients-count">{clients.length} active</span>
        </div>
        {recentClients.length > 0 ? (
          <div className="dash-clients-list">
            {recentClients.map(c => (
              <div key={c.mac} className="dash-client-row">
                <div className="dash-client-dot" />
                <span className="dash-client-name">{c.name}</span>
                <span className="dash-client-speed mono">{c.speed.toFixed(1)} Mbps</span>
                <span className="dash-client-data mono">{c.dataShared.toFixed(1)} MB</span>
              </div>
            ))}
          </div>
        ) : (
          <div className="dash-clients-empty">
            {isOnline ? 'No devices connected yet' : 'Enable sharing to accept connections'}
          </div>
        )}
      </div>

    </div>
  );
}
