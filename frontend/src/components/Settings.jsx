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
      setMessage('Configuration saved successfully.');
      setTimeout(() => setMessage(''), 3000);
    } catch (err) {
      setMessage('Error saving configuration.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="settings-wrap" onSubmit={handleSubmit}>
      {/* Node Identity */}
      <div className="form-group">
        <label className="form-label" htmlFor="displayName">Node Display Name</label>
        <input 
          id="displayName"
          type="text" 
          className="form-input" 
          value={displayName} 
          onChange={(e) => setDisplayName(e.target.value)} 
          required 
          placeholder="e.g. Node-Alpha-7"
        />
        <span className="form-desc">Provide a recognizable name for your sharing node in the Zoop Network.</span>
      </div>

      {/* Visibility */}
      <div className="form-group">
        <label className="form-label">Network Visibility</label>
        <div className="radio-group">
          <div 
            className={`radio-card ${isPublic ? 'active' : ''}`}
            onClick={() => setIsPublic(true)}
          >
            <strong>Public</strong>
            <span>Discoverable by any nearby client looking for connections.</span>
          </div>
          <div 
            className={`radio-card ${!isPublic ? 'active' : ''}`}
            onClick={() => setIsPublic(false)}
          >
            <strong>Private</strong>
            <span>Restricted only to trusted cryptographic node identities.</span>
          </div>
        </div>
      </div>

      {/* Bandwidth Cap Slider */}
      <div className="form-group">
        <div className="form-label-wrap">
          <label className="form-label" htmlFor="bandwidthSlider">Bandwidth Cap</label>
          <span className="range-value">{bandwidthCapPercent}%</span>
        </div>
        <div className="range-container">
          <input 
            id="bandwidthSlider"
            type="range" 
            min="10" 
            max="100" 
            className="range-input" 
            value={bandwidthCapPercent} 
            onChange={(e) => setBandwidthCapPercent(e.target.value)}
          />
        </div>
        <span className="form-desc">Enforces a percentage limit of your network speed dedicated to guest routing.</span>
      </div>

      {/* Concurrent Clients */}
      <div className="form-group">
        <label className="form-label" htmlFor="maxClients">Maximum Clients</label>
        <input 
          id="maxClients"
          type="number" 
          min="1" 
          max="50"
          className="form-input" 
          value={maxClients} 
          onChange={(e) => setMaxClients(e.target.value)} 
          required 
        />
        <span className="form-desc">Set the max limit of concurrent client devices allowed to share your node.</span>
      </div>

      {/* Network Interface Selection */}
      <div className="form-group">
        <label className="form-label" htmlFor="networkInterface">Preferred Network Interface</label>
        <select 
          id="networkInterface"
          className="form-select" 
          value={preferredInterface} 
          onChange={(e) => setPreferredInterface(e.target.value)}
        >
          {interfaces.map((name) => (
            <option key={name} value={name}>{name}</option>
          ))}
        </select>
        <span className="form-desc">The hardware interface used to create the guest routing hotspot.</span>
      </div>

      {/* Auto-Start */}
      <div className="toggle-row">
        <div>
          <label className="form-label" style={{ display: 'block' }}>System Auto-Start</label>
          <span className="form-desc">Launch the sharing node daemon automatically during computer boot.</span>
        </div>
        <label className="switch">
          <input 
            type="checkbox" 
            checked={autoStartOnBoot} 
            onChange={(e) => setAutoStartOnBoot(e.target.checked)} 
          />
          <span className="slider"></span>
        </label>
      </div>

      {/* Actions and Status messages */}
      <div className="settings-actions" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '13px', color: 'var(--green)', fontWeight: '600' }}>
          {message}
        </span>
        <button 
          type="submit" 
          className="btn btn-primary" 
          disabled={saving}
        >
          <Save size={16} />
          <span>{saving ? 'Saving...' : 'Save Settings'}</span>
        </button>
      </div>
    </form>
  );
}
