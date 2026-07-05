import { useState, useEffect, useCallback } from 'react';
import * as runtime from "../wailsjs/runtime/runtime";
import { GetConfig, SaveConfig, ToggleSharing, GetInterfaces, DisconnectClient, GetLogs, GetTunnelStatus, DiscoverPeers, ConnectToPeer, DisconnectFromPeer, GetPlatform } from "../wailsjs/go/main/NodeService";
import { toast } from './components/Toast';

export function useTheme() {
  const [theme, setTheme] = useState(() => localStorage.getItem('zoop-theme') || 'dark');

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('zoop-theme', theme);
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme(t => t === 'light' ? 'dark' : 'light');
  }, []);

  return [theme, toggleTheme];
}

export function useConfig() {
  const [config, setConfig] = useState(null);
  const [eulaVisible, setEulaVisible] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    GetConfig()
      .then(cfg => {
        setConfig(cfg);
        return GetPlatform().then(platform => ({ cfg, platform }));
      })
      .then(({ cfg, platform }) => {
        if (!cfg.system?.eulaAccepted && platform === 'windows') {
          setEulaVisible(true);
        }
      })
      .catch(err => {
        console.error(err);
        toast.error('Unable to load configuration');
      })
      .finally(() => setLoading(false));
  }, []);

  const acceptEula = useCallback(() => {
    if (!config) return;
    const updated = { ...config };
    if (!updated.system) updated.system = {};
    updated.system.eulaAccepted = true;
    SaveConfig(updated).then(() => {
      setConfig(updated);
      setEulaVisible(false);
      toast.info('Thank you for accepting the EULA.');
    }).catch(err => {
      console.error(err);
      toast.error('Failed to persist EULA acceptance');
    });
  }, [config]);

  const handleSaveConfig = useCallback((updatedConfig) => {
    return SaveConfig(updatedConfig)
      .then(() => {
        setConfig(updatedConfig);
        toast.success('Settings saved successfully');
      })
      .catch(err => {
        console.error(err);
        toast.error(err?.message || 'Failed to save settings');
        throw err;
      });
  }, []);

  return { config, setConfig, loading, eulaVisible, acceptEula, handleSaveConfig };
}

export function useTelemetry() {
  const [telemetry, setTelemetry] = useState({
    uploadSpeed: 0, downloadSpeed: 0,
    activeClientsCount: 0, totalDataShared: 0, sharingEnabled: false,
    tunnelActive: false, tunnelError: '', startedAt: '', publicIP: ''
  });
  const [bandwidthHistory, setBandwidthHistory] = useState(Array(20).fill(0));
  const [dataHistory, setDataHistory] = useState(Array(20).fill(0));
  const [uptimeString, setUptimeString] = useState('0d 0h 0m');

  useEffect(() => {
    const updateUptime = () => {
      const startedAt = Date.parse(telemetry.startedAt);
      if (!telemetry.startedAt || Number.isNaN(startedAt) || !telemetry.tunnelActive) {
        setUptimeString('0d 0h 0m');
        return;
      }
      const diff = Math.max(0, Date.now() - startedAt);
      const d = Math.floor(diff / (24 * 3600e3));
      const h = Math.floor((diff % (24 * 3600e3)) / 3600e3);
      const m = Math.floor((diff % 3600e3) / 60e3);
      setUptimeString(`${d}d ${h}h ${m}m`);
    };
    updateUptime();
    const interval = setInterval(updateUptime, 5000);
    return () => clearInterval(interval);
  }, [telemetry.startedAt, telemetry.tunnelActive]);

  useEffect(() => {
    const unsubscribe = [];
    unsubscribe.push(runtime.EventsOn('telemetry_updated', (data) => {
      setTelemetry(data);
      const totalSpeed = data.uploadSpeed + data.downloadSpeed;
      setBandwidthHistory(prev => data.tunnelActive ? [...prev.slice(1), totalSpeed] : Array(20).fill(0));
      setDataHistory(prev => data.tunnelActive ? [...prev.slice(1), data.totalDataShared] : Array(20).fill(0));
    }));
    return () => unsubscribe.forEach(fn => fn());
  }, []);

  useEffect(() => {
    GetTunnelStatus()
      .then(status => {
        if (status) setTelemetry(prev => ({ ...prev, ...status }));
      })
      .catch(err => console.error(err));
  }, []);

  return { telemetry, setTelemetry, bandwidthHistory, dataHistory, uptimeString };
}

export function usePeers() {
  const [discoveredPeers, setDiscoveredPeers] = useState([]);
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [connectedPeerId, setConnectedPeerId] = useState(null);

  const loadDiscoveredPeers = useCallback(() => {
    setDiscoveryLoading(true);
    DiscoverPeers()
      .then(setDiscoveredPeers)
      .catch(err => {
        console.error(err);
        setDiscoveredPeers([]);
        toast.warning('Unable to discover network devices.');
      })
      .finally(() => setDiscoveryLoading(false));
  }, []);

  const handleConnectToPeer = useCallback(async (nodeId) => {
    try {
      await ConnectToPeer(nodeId);
      setConnectedPeerId(nodeId);
      toast.success('Connected to peer successfully.');
    } catch (err) {
      console.error(err);
      toast.error('Failed to connect to peer.');
      throw err;
    }
  }, []);

  const handleDisconnectFromPeer = useCallback(async () => {
    try {
      await DisconnectFromPeer();
      setConnectedPeerId(null);
      toast.success('Disconnected from peer.');
    } catch (err) {
      console.error(err);
      toast.error('Failed to disconnect from peer.');
    }
  }, []);

  return { discoveredPeers, discoveryLoading, connectedPeerId, loadDiscoveredPeers, handleConnectToPeer, handleDisconnectFromPeer };
}

export function useClients() {
  const [clients, setClients] = useState([]);

  useEffect(() => {
    const unsub = runtime.EventsOn('clients_updated', (data) => setClients(data || []));
    return () => unsub();
  }, []);

  const handleDisconnectClient = useCallback((mac) => {
    DisconnectClient(mac)
      .then(success => {
        if (success) {
          setClients(prev => prev.filter(c => c.mac !== mac));
          toast.success('Client disconnected successfully.');
        } else {
          toast.warning('Client was not found or already disconnected.');
        }
      })
      .catch(err => {
        console.error(err);
        toast.error('Failed to disconnect client.');
      });
  }, []);

  return { clients, handleDisconnectClient };
}

export function useLogs() {
  const [logs, setLogs] = useState([]);

  useEffect(() => {
    GetLogs()
      .then(l => setLogs(l || []))
      .catch(err => {
        console.error(err);
        toast.warning('Unable to load logs');
      });

    const unsub = runtime.EventsOn('log_added', (newLog) => {
      setLogs(prev => [...prev, newLog].slice(-100));
    });
    return () => unsub();
  }, []);

  return logs;
}

export function useInterfaces() {
  const [interfaces, setInterfaces] = useState([]);

  useEffect(() => {
    GetInterfaces()
      .then(setInterfaces)
      .catch(err => {
        console.error(err);
        toast.warning('Unable to detect network interfaces');
      });
  }, []);

  return interfaces;
}

export function useSharing(setTelemetry) {
  const [wgInfo, setWgInfo] = useState(null);
  const [wgLoading, setWgLoading] = useState(false);

  useEffect(() => {
    const unsubs = [];
    unsubs.push(runtime.EventsOn('wg_tunnel_ready', (info) => {
      setWgLoading(false);
      setWgInfo(info);
    }));
    unsubs.push(runtime.EventsOn('wg_tunnel_error', (info) => {
      setWgLoading(false);
      setWgInfo(info);
      toast.error(info?.error || 'WireGuard tunnel failed to start');
    }));
    unsubs.push(runtime.EventsOn('wg_tunnel_stopped', () => {
      setWgLoading(false);
      setWgInfo(null);
    }));
    return () => unsubs.forEach(fn => fn());
  }, []);

  const handleToggleSharing = useCallback((nextState) => {
    if (nextState) {
      setWgLoading(true);
      setWgInfo(null);
    } else {
      setWgInfo(null);
      setWgLoading(false);
    }

    ToggleSharing(nextState)
      .then(result => {
        if (!nextState) {
          if (setTelemetry) {
            setTelemetry(prev => ({ ...prev, sharingEnabled: result, tunnelActive: result }));
          }
          toast.info('Sharing disabled successfully.');
        }
      })
      .catch(err => {
        console.error(err);
        setWgLoading(false);
        toast.error('Unable to change sharing state.');
      });
  }, [setTelemetry]);

  return { wgInfo, wgLoading, handleToggleSharing };
}
