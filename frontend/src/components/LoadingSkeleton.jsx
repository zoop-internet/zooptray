import React from 'react';

export function SkeletonLine({ width }) {
  return <div className="skeleton skeleton-line" style={{ width }} />;
}

export function SkeletonTitle({ width }) {
  return <div className="skeleton skeleton-title" style={{ width }} />;
}

export function SkeletonCard() {
  return <div className="skeleton skeleton-card" />;
}

export function DashboardSkeleton() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div className="node-status-card" style={{ border: '1px solid var(--line)' }}>
        <div style={{ display: 'flex', gap: '24px', alignItems: 'center' }}>
          <div className="skeleton" style={{ width: 96, height: 96, borderRadius: '50%' }} />
          <div style={{ flex: 1 }}>
            <SkeletonTitle width="40%" />
            <SkeletonLine width="80%" />
            <div style={{ height: 12 }} />
            <div className="skeleton" style={{ width: 140, height: 40, borderRadius: 9 }} />
          </div>
        </div>
      </div>
      <div className="telemetry-grid">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </div>
    </div>
  );
}

export function SettingsSkeleton() {
  return (
    <div className="settings-layout">
      {[1, 2, 3].map(i => (
        <div key={i} className="settings-card">
          <SkeletonTitle width="35%" />
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <SkeletonLine width="90%" />
            <SkeletonLine width="70%" />
            <SkeletonLine width="50%" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function DevicesSkeleton() {
  return (
    <div className="devices-layout">
      <div className="devices-header-bar">
        <SkeletonLine width="120px" />
        <SkeletonLine width="200px" />
      </div>
      <div className="devices-grid">
        {[1, 2].map(i => (
          <div key={i} className="device-card">
            <SkeletonCard />
          </div>
        ))}
      </div>
    </div>
  );
}
