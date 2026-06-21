import React, { useState, useEffect } from 'react';
import { Save, User, Network, Settings2, Power } from 'lucide-react';
import Dropdown from './Dropdown';
import { toast } from './Toast';

export default function Settings({ config, interfaces, onSaveConfig }) {
  const [displayName, setDisplayName]           = useState('');
  const [isPublic, setIsPublic]                 = useState(true);
  const [maxClients, setMaxClients]             = useState(10);
  const [bandwidthCapPercent, setBandwidthCap]  = useState(68);
  const [routingPolicy, setRoutingPolicy]       = useState('balanced');
  const [preferredInterface, setInterface]       = useState('');
  const [autoStart, setAutoStart]         = useState(true);
  const [heartbeatInterval, setHeartbeat]       = useState(15);
  const [saving, setSaving]                     = useState(false);

  const routingOptions = [
    { label: 'Balanced',     value: 'balanced'     },
    { label: 'Performance',  value: 'performance'  },
    { label: 'Low Latency',  value: 'low-latency'  },
    { label: 'Conservative', value: 'conservative' },
  ];

  const heartbeatOptions = [
    { label: '5 seconds',  value: '5'  },
    { label: '10 seconds', value: '10' },
    { label: '15 seconds', value: '15' },
    { label: '30 seconds', value: '30' },
    { label: '60 seconds', value: '60' },
  ];

  const ifaceOptions = interfaces.map(i => ({ label: i, value: i }));

  useEffect(() => {
    if (config) {
      setDisplayName(config.node.displayName || 'Node-Alpha-7');
      setIsPublic(config.node.isPublic !== false);
      setMaxClients(config.sharing.maxClients || 10);
      setBandwidthCap(config.sharing.bandwidthCapPercent || 68);
      setRoutingPolicy(config.sharing.routingPolicy || 'balanced');
      setInterface(config.system.preferredInterface || (interfaces[0] || 'wlan0'));
      setAutoStart(config.system.autoStartOnBoot !== false);
      setHeartbeat(config.api.heartbeatIntervalSeconds || 15);
    }
  }, [config, interfaces]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);

    const updatedConfig = {
      node: {
        id: displayName.toLowerCase().replace(/[^a-z0-9]/g, '-'),
        displayName,
        privateKeyPath: config.node.privateKeyPath,
        isPublic,
      },
      sharing: {
        enabled: config.sharing.enabled,
        maxClients: parseInt(maxClients, 10),
        bandwidthCapPercent: parseInt(bandwidthCapPercent, 10),
        routingPolicy,
      },
      system: {
        autoStartOnBoot: autoStart,
        preferredInterface,
      },
      api: {
        endpoint: config.api.endpoint,
        heartbeatIntervalSeconds: parseInt(heartbeatInterval, 10),
      },
    };

    try {
      await onSaveConfig(updatedConfig);
      toast.success('Settings saved successfully');
    } catch {
      toast.error('Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="settings-layout" onSubmit={handleSubmit}>

      {/* Node Identity */}
      <div className="settings-card">
        <h3><User size={15} style={{ marginRight: 8, verticalAlign: 'middle' }} />Node Identity</h3>

        <div className="settings-group">
          <label htmlFor="displayName">Display Name</label>
          <input id="displayName" type="text" className="settings-input"
            value={displayName} onChange={e => setDisplayName(e.target.value)}
            required placeholder="e.g. Node-Alpha-7"
          />
          <span className="settings-desc">Public label for your node shown in the Zoop Network dashboard.</span>
        </div>

        <div className="settings-group">
          <label>Network Visibility</label>
          <div className="visibility-options">
            <div className={`visibility-card ${isPublic ? 'active' : ''}`} onClick={() => setIsPublic(true)}>
              <strong>Public Node</strong>
              <span>Discoverable by any nearby guest device looking for a gateway.</span>
            </div>
            <div className={`visibility-card ${!isPublic ? 'active' : ''}`} onClick={() => setIsPublic(false)}>
              <strong>Private Node</strong>
              <span>Accessible only to cryptographically trusted client credentials.</span>
            </div>
          </div>
        </div>
      </div>

      {/* Network Configuration */}
      <div className="settings-card">
        <h3><Network size={15} style={{ marginRight: 8, verticalAlign: 'middle' }} />Network Configuration</h3>

        <div className="settings-group">
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <label htmlFor="bandwidthSlider">Bandwidth Cap Allocation</label>
            <span className="range-display-val">{bandwidthCapPercent}%</span>
          </div>
          <div className="slider-container">
            <input id="bandwidthSlider" type="range" min="10" max="100"
              className="custom-range" value={bandwidthCapPercent}
              onChange={e => setBandwidthCap(parseInt(e.target.value, 10))}
            />
          </div>
          <span className="settings-desc">Percentage of your total bandwidth to share with guest clients.</span>
        </div>

        <div className="settings-group">
          <label htmlFor="maxClients">Maximum Concurrent Clients</label>
          <input id="maxClients" type="number" min="1" max="50"
            className="settings-input" value={maxClients}
            onChange={e => setMaxClients(e.target.value)} required
            style={{ maxWidth: '160px' }}
          />
          <span className="settings-desc">Hard limit on how many guest devices can connect simultaneously.</span>
        </div>

        <div className="settings-group">
          <label>Routing Policy</label>
          <Dropdown
            value={routingPolicy}
            onChange={setRoutingPolicy}
            options={routingOptions}
          />
          <span className="settings-desc">
            <strong>Balanced</strong> — even distribution. <strong>Performance</strong> — maximize speed. <strong>Low Latency</strong> — minimize delay.
          </span>
        </div>

        <div className="settings-group">
          <label>Preferred Network Interface</label>
          <Dropdown
            value={preferredInterface}
            onChange={setInterface}
            options={ifaceOptions.length > 0 ? ifaceOptions : [{ label: preferredInterface || 'wlan0', value: preferredInterface || 'wlan0' }]}
          />
          <span className="settings-desc">The hardware interface used to create the guest routing hotspot.</span>
        </div>
      </div>

      {/* Advanced */}
      <div className="settings-card">
        <h3><Settings2 size={15} style={{ marginRight: 8, verticalAlign: 'middle' }} />Advanced</h3>

        <div className="settings-group">
          <label>Heartbeat Interval</label>
          <Dropdown
            value={String(heartbeatInterval)}
            onChange={v => setHeartbeat(parseInt(v, 10))}
            options={heartbeatOptions}
          />
          <span className="settings-desc">How often to send a keepalive ping to the Zoop coordinator API. Lower = more responsive, higher = less network overhead.</span>
        </div>

        <div className="settings-group">
          <label>API Endpoint</label>
          <input type="text" className="settings-input"
            value={config?.api?.endpoint || 'https://api.zoopnetwork.com/v1'}
            readOnly
            style={{ opacity: 0.6, cursor: 'not-allowed' }}
          />
          <span className="settings-desc">Central coordinator endpoint (read-only in this release).</span>
        </div>
      </div>

      {/* Startup */}
      <div className="settings-card">
        <h3><Power size={15} style={{ marginRight: 8, verticalAlign: 'middle' }} />Startup</h3>

        <div className="toggle-switch-row">
          <div>
            <label style={{ display: 'block', marginBottom: '4px' }}>Launch on System Start</label>
            <span className="settings-desc">
              Automatically start the sharing daemon when your computer boots.
              {' '}
              <span style={{ color: 'var(--muted-dark)', fontSize: '12px' }}>
                (Linux: writes to ~/.config/autostart/)
              </span>
            </span>
          </div>
          <label className="toggle-switch">
            <input type="checkbox" checked={autoStart} onChange={e => setAutoStart(e.target.checked)} />
            <span className="toggle-slider"></span>
          </label>
        </div>
      </div>

      {/* Save footer */}
      <div className="settings-footer">
        <span style={{ fontSize: '13px', color: 'var(--muted-dark)' }}>
          Changes take effect immediately after saving.
        </span>
        <button type="submit" className="btn btn-primary" disabled={saving} style={{ minWidth: '140px' }}>
          <Save size={15} />
          <span>{saving ? 'Saving…' : 'Save Settings'}</span>
        </button>
      </div>
    </form>
  );
}
