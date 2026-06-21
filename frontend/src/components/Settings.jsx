import React, { useState, useEffect } from 'react';
import { Save } from 'lucide-react';

export default function Settings({ config, interfaces, onSaveConfig }) {
  const [displayName, setDisplayName] = useState('');
  const [isPublic, setIsPublic] = useState(true);
  const [maxClients, setMaxClients] = useState(10);
  const [bandwidthCapPercent, setBandwidthCapPercent] = useState(68);
  const [preferredInterface, setPreferredInterface] = useState('');
  const [autoStartOnBoot, setAutoStartOnBoot] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  // Sync state with config when it loads
  useEffect(() => {
    if (config) {
      setDisplayName(config.node.displayName || 'Node-Alpha-7');
      setIsPublic(config.node.isPublic !== false);
      setMaxClients(config.sharing.maxClients || 10);
      setBandwidthCapPercent(config.sharing.bandwidthCapPercent || 68);
      setPreferredInterface(config.system.preferredInterface || 'wlan0');
      setAutoStartOnBoot(config.system.autoStartOnBoot !== false);
    }
  }, [config]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMessage('');

    // Construct updated config matching Go struct
    const updatedConfig = {
      node: {
        id: displayName.toLowerCase().replace(/[^a-z0-9]/g, '-'),
        displayName: displayName,
        privateKeyPath: config.node.privateKeyPath,
        isPublic: isPublic
      },
      sharing: {
        enabled: config.sharing.enabled,
        maxClients: parseInt(maxClients, 10),
        bandwidthCapPercent: parseInt(bandwidthCapPercent, 10),
        routingPolicy: config.sharing.routingPolicy
      },
      system: {
        autoStartOnBoot: autoStartOnBoot,
        preferredInterface: preferredInterface
      },
      api: config.api
    };

    try {
      await onSaveConfig(updatedConfig);
      setMessage('Configuration saved.');
      setTimeout(() => setMessage(''), 3000);
    } catch (err) {
      setMessage('Error saving config.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="settings-layout" onSubmit={handleSubmit}>
      {/* Node Profile Settings */}
      <div className="settings-card">
        <h3>Node Identity</h3>

        <div className="settings-group">
          <label htmlFor="displayName">Node Display Name</label>
          <input 
            id="displayName"
            type="text" 
            className="settings-input" 
            value={displayName} 
            onChange={(e) => setDisplayName(e.target.value)} 
            required 
            placeholder="e.g. Node-Alpha-7"
          />
          <span className="settings-desc">Provide a recognizable name for your sharing node in the Zoop Network.</span>
        </div>

        <div className="settings-group">
          <label>Network Visibility</label>
          <div className="visibility-options">
            <div 
              className={`visibility-card ${isPublic ? 'active' : ''}`}
              onClick={() => setIsPublic(true)}
            >
              <strong>Public Node</strong>
              <span>Discoverable by any nearby client looking for hotspot gateways.</span>
            </div>
            <div 
              className={`visibility-card ${!isPublic ? 'active' : ''}`}
              onClick={() => setIsPublic(false)}
            >
              <strong>Private Node</strong>
              <span>Restricted only to trusted cryptographic client credentials.</span>
            </div>
          </div>
        </div>
      </div>

      {/* Network Allocations */}
      <div className="settings-card">
        <h3>Network & Hardware Configuration</h3>

        {/* Bandwidth Cap Slider */}
        <div className="settings-group">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <label htmlFor="bandwidthSlider">Bandwidth Cap Allocation</label>
            <span className="range-display-val">{bandwidthCapPercent}%</span>
          </div>
          <div className="slider-container">
            <input 
              id="bandwidthSlider"
              type="range" 
              min="10" 
              max="100" 
              className="custom-range" 
              value={bandwidthCapPercent} 
              onChange={(e) => setBandwidthCapPercent(parseInt(e.target.value, 10))}
            />
          </div>
          <span className="settings-desc">Enforces a percentage limit of your network speed dedicated to guest routing.</span>
        </div>

        {/* Concurrent Clients */}
        <div className="settings-group">
          <label htmlFor="maxClients">Maximum Allowed Clients</label>
          <input 
            id="maxClients"
            type="number" 
            min="1" 
            max="50"
            className="settings-input" 
            value={maxClients} 
            onChange={(e) => setMaxClients(e.target.value)} 
            required 
          />
          <span className="settings-desc">Set the max limit of concurrent client devices allowed to share your node.</span>
        </div>

        {/* Interface select */}
        <div className="settings-group">
          <label htmlFor="networkInterface">Hardware Network Interface</label>
          <select 
            id="networkInterface"
            className="settings-input" 
            value={preferredInterface} 
            onChange={(e) => setPreferredInterface(e.target.value)}
            style={{ cursor: 'pointer' }}
          >
            {interfaces.map((name) => (
              <option key={name} value={name}>{name}</option>
            ))}
          </select>
          <span className="settings-desc">The hardware interface used to create the guest routing hotspot.</span>
        </div>
      </div>

      {/* Startup Policies */}
      <div className="settings-card">
        <h3>Startup Policies</h3>
        
        <div className="toggle-switch-row">
          <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
            <label style={{ cursor: 'pointer' }}>System Auto-Start</label>
            <span className="settings-desc">Launch the sharing node daemon automatically during computer boot.</span>
          </div>
          <label className="toggle-switch">
            <input 
              type="checkbox" 
              checked={autoStartOnBoot} 
              onChange={(e) => setAutoStartOnBoot(e.target.checked)} 
            />
            <span className="toggle-slider"></span>
          </label>
        </div>
      </div>

      {/* Footer saving indicator and save button */}
      <div className="settings-footer">
        <span className="settings-msg">{message}</span>
        <button 
          type="submit" 
          className="btn btn-primary" 
          disabled={saving}
          style={{ minWidth: '130px' }}
        >
          <Save size={14} />
          <span>{saving ? 'Saving...' : 'Save Settings'}</span>
        </button>
      </div>
    </form>
  );
}
