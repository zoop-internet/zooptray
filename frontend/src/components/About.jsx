import React, { useState, useEffect, useRef } from 'react';
import zoopLogo from '../assets/images/zoopicon.png';
import { Terminal, ChevronDown, ChevronUp, Cpu, HardDrive } from 'lucide-react';

export default function About({ config, logs }) {
  const [filter, setFilter] = useState('all');
  const [logsExpanded, setLogsExpanded] = useState(true);
  const logEndRef = useRef(null);

  // Auto-scroll to bottom of logs on update when open
  useEffect(() => {
    if (logsExpanded) {
      logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, logsExpanded]);

  const filteredLogs = logs.filter(log => {
    if (filter === 'all') return true;
    return log.level === filter;
  });

  return (
    <div className="about-layout">
      {/* Brand card */}
      <div className="about-card">
        <div className="about-logo-box">
          <img src={zoopLogo} alt="Zoop Logo" />
        </div>
        <div className="about-info-text">
          <h2>Zoop Desktop Node</h2>
          <p style={{ marginBottom: '6px' }}>
            <strong>Version 1.0.0</strong> — Built by the Zoop Network team.
          </p>
          <p style={{ fontSize: '12px' }}>
            A lightweight daemon that securely routes guest network traffic. By running this node, your device operates as a secure gateway for decentralised access.
          </p>
        </div>
      </div>

      {/* Node spec credentials */}
      <div className="speed-limits-card">
        <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px', borderBottom: '1px solid var(--line)', paddingBottom: '10px' }}>
          <Cpu size={16} />
          <span>System Information</span>
        </h3>
        
        <div className="form-row">
          <div className="form-row-left">
            <span className="form-row-label">Node Identifier</span>
            <span className="form-row-desc">The cryptographic key ID registered with the network.</span>
          </div>
          <span className="meta-value">{config?.node.id || 'loading...'}</span>
        </div>

        <div className="form-row">
          <div className="form-row-left">
            <span className="form-row-label">Private Key Storage</span>
            <span className="form-row-desc">Path containing local credentials.</span>
          </div>
          <span className="meta-value" style={{ fontSize: '11px' }} title={config?.node.privateKeyPath}>
            {config?.node.privateKeyPath ? `${config.node.privateKeyPath.slice(0, 32)}...` : 'loading...'}
          </span>
        </div>

        <div className="form-row">
          <div className="form-row-left">
            <span className="form-row-label">Discovery Mode</span>
            <span className="form-row-desc">How peers discover each other on the network.</span>
          </div>
          <span className="meta-value" style={{ fontSize: '11px' }}>
            {config?.p2p?.enabled ? 'P2P DHT Mesh (libp2p)' : 'Cloudflare Worker'}
          </span>
        </div>
      </div>

      {/* Collapsible Diagnostic Logs */}
      <div className="logs-card">
        <div 
          className="logs-card-header" 
          onClick={() => setLogsExpanded(prev => !prev)}
        >
          <div className="logs-card-header-left">
            <Terminal size={16} />
            <span>Diagnostic Logs Stream</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {logsExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
          </div>
        </div>

        {logsExpanded && (
          <>
            {/* Header filters */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 16px', background: 'var(--surface-soft)', borderBottom: '1px solid var(--line)' }}>
              <span style={{ fontSize: '11px', color: 'var(--muted)', fontWeight: '600' }}>CONSOLE LOG STREAM</span>
              <div className="logs-filters">
                {['all', 'info', 'warning', 'error'].map((level) => (
                  <button
                    key={level}
                    type="button"
                    className={`log-filter-btn ${filter === level ? 'active' : ''}`}
                    onClick={(e) => {
                      e.stopPropagation();
                      setFilter(level);
                    }}
                  >
                    {level.toUpperCase()}
                  </button>
                ))}
              </div>
            </div>

            {/* Log view lines */}
            <div className="log-stream">
              {filteredLogs.length === 0 ? (
                <div style={{ padding: '20px 0', textAlign: 'center', color: 'var(--muted-dark)' }}>
                  No diagnostic events matching this level.
                </div>
              ) : (
                filteredLogs.map((log, index) => (
                  <div key={index} className={`log-row ${log.level}`}>
                    <span className="log-row-time">{log.timestamp}</span>
                    <span className="log-row-badge">{log.level}</span>
                    <span className="log-row-msg">{log.message}</span>
                  </div>
                ))
              )}
              <div ref={logEndRef} />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
