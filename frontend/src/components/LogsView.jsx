import React, { useState, useEffect, useRef } from 'react';
import { Terminal } from 'lucide-react';

export default function LogsView({ logs }) {
  const [filter, setFilter] = useState('all'); // 'all', 'info', 'warning', 'error'
  const logEndRef = useRef(null);

  // Auto-scroll to bottom of logs on update
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  const filteredLogs = logs.filter(log => {
    if (filter === 'all') return true;
    return log.level === filter;
  });

  return (
    <div>
      <div className="logs-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--muted)' }}>
          <Terminal size={18} />
          <span style={{ fontSize: '13.5px', fontWeight: '600' }}>Local Daemon Output</span>
        </div>
        <div className="logs-filters">
          {['all', 'info', 'warning', 'error'].map((level) => (
            <button
              key={level}
              type="button"
              className={`log-filter-btn ${filter === level ? 'active' : ''}`}
              onClick={() => setFilter(level)}
            >
              {level.toUpperCase()}
            </button>
          ))}
        </div>
      </div>

      <div className="log-list">
        {filteredLogs.length === 0 ? (
          <div className="empty-state" style={{ padding: '30px 0' }}>
            No logs matching this filter criteria.
          </div>
        ) : (
          filteredLogs.map((log, index) => (
            <div key={index} className={`log-item ${log.level}`}>
              <span className="log-time">{log.timestamp}</span>
              <span className="log-badge">{log.level}</span>
              <span className="log-msg">{log.message}</span>
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
    </div>
  );
}
