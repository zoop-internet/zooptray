import React from 'react';
import { Wallet, ExternalLink, TrendingUp, Clock, Award } from 'lucide-react';
import * as runtime from "../../wailsjs/runtime/runtime";

export default function Earnings() {
  const handleLearnMore = () => {
    runtime.BrowserOpenURL("https://zoopnetwork.com");
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', maxWidth: '1000px', margin: '0 auto', width: '100%' }}>
      <div className="earnings-card">
        <div className="earnings-icon-container">
          <Wallet size={32} />
        </div>
        <h2>Earnings coming soon</h2>
        <p>
          We're working on bringing you detailed earnings and payment insights directly inside the desktop dashboard.
        </p>
        <button 
          type="button" 
          className="btn btn-primary"
          onClick={handleLearnMore}
        >
          <span>Learn more</span>
          <ExternalLink size={14} />
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '20px' }}>
        {[
          { icon: <TrendingUp size={22} />, label: 'Estimated Earnings', value: '—', accent: 'var(--green)' },
          { icon: <Clock size={22} />, label: 'Total Uptime Rewards', value: '—', accent: 'var(--blue)' },
          { icon: <Award size={22} />, label: 'Network Contribution', value: '—', accent: 'var(--purple)' },
        ].map(item => (
          <div key={item.label} className="telemetry-card" style={{ borderTop: `3px solid ${item.accent}` }}>
            <div className="telemetry-header">
              <span className="telemetry-label">{item.label}</span>
              <div className="telemetry-value-wrap" style={{ marginTop: '8px', alignItems: 'center', gap: '10px' }}>
                <span style={{ color: item.accent }}>{item.icon}</span>
                <span className="telemetry-value" style={{ fontFamily: 'var(--font)', fontSize: '22px' }}>{item.value}</span>
              </div>
            </div>
            <div style={{ fontSize: '12px', color: 'var(--muted-dark)', marginTop: '8px' }}>
              Available after mainnet launch. Stay tuned.
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
