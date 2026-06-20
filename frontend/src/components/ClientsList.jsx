import React from 'react';
import { Wifi, XCircle } from 'lucide-react';

export default function ClientsList({ clients, sharingEnabled, onDisconnectClient }) {
  if (!sharingEnabled) {
    return (
      <div className="empty-state">
        <Wifi size={40} style={{ opacity: 0.3 }} />
        <h3>Hotspot is Offline</h3>
        <p style={{ color: 'var(--muted)', marginTop: '6px' }}>
          Enable sharing on the Dashboard to allow guest clients to connect.
        </p>
      </div>
    );
  }

  if (clients.length === 0) {
    return (
      <div className="empty-state">
        <Wifi size={40} style={{ opacity: 0.3 }} />
        <h3>No Clients Connected</h3>
        <p style={{ color: 'var(--muted)', marginTop: '6px' }}>
          Waiting for nearby guest devices to discover your node and connect...
        </p>
      </div>
    );
  }

  return (
    <div className="table-wrap">
      <table className="ztable">
        <thead>
          <tr>
            <th>Client Device</th>
            <th>IP Address</th>
            <th>Connection Time</th>
            <th>Current Speed</th>
            <th>Data Transferred</th>
            <th style={{ textAlign: 'right' }}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {clients.map((client) => (
            <tr key={client.mac}>
              <td>
                <div className="client-device">
                  <span className="client-name">{client.name}</span>
                  <span className="client-mac">{client.mac}</span>
                </div>
              </td>
              <td>
                <span className="client-ip">{client.ip}</span>
              </td>
              <td>
                <span className="client-time">{client.connectedAt}</span>
              </td>
              <td>
                <span className="client-speed">{client.speed.toFixed(1)} Mbps</span>
              </td>
              <td>
                <span className="client-data">{client.dataShared.toFixed(1)} MB</span>
              </td>
              <td style={{ textAlign: 'right' }}>
                <button 
                  className="btn btn-danger btn-sm"
                  onClick={() => onDisconnectClient(client.mac)}
                  title="Disconnect and Block Client"
                >
                  <XCircle size={14} />
                  <span>Kick</span>
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
