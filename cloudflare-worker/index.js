export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const { pathname } = url;
    const method = request.method.toUpperCase();

    if (pathname === '/register' && method === 'POST') {
      return handleRegister(request, env);
    }

    if (pathname === '/discover' && method === 'GET') {
      return handleDiscover(request, env);
    }

    if (pathname === '/heartbeat' && method === 'POST') {
      return handleHeartbeat(request, env);
    }

    if (pathname === '/register' && method === 'DELETE') {
      return handleUnregister(request, env);
    }

    if (pathname === '/connect-request' && method === 'POST') {
      return handleConnectRequest(request, env);
    }

    if (pathname === '/connect-request' && method === 'DELETE') {
      return handleCancelConnect(request, env);
    }

    if (pathname === '/pending-peers' && method === 'GET') {
      return handlePendingPeers(request, env);
    }

    if (pathname === '/disconnect-notification' && method === 'POST') {
      return handleDisconnectNotification(request, env);
    }

    if (pathname === '/disconnect-notifications' && method === 'GET') {
      return handleDisconnectNotifications(request, env);
    }

    if (pathname === '/relay-request' && method === 'GET') {
      return handleRelayRequest(request, env);
    }

    if (pathname === '/download' && method === 'GET') {
      return handleDownload(request, env);
    }

    return jsonResponse({ error: 'Not Found' }, 404);
  },
};

const DEFAULT_TTL_SECONDS = 60;

async function getEntry(nodeId, env) {
  if (env.DISCOVERY_KV) {
    const data = await env.DISCOVERY_KV.get(nodeId);
    if (!data) return null;
    try {
      return JSON.parse(data);
    } catch {
      return null;
    }
  }
  return null;
}

async function putEntry(nodeId, entry, env) {
  if (env.DISCOVERY_KV) {
    await env.DISCOVERY_KV.put(nodeId, JSON.stringify(entry), { expirationTtl: DEFAULT_TTL_SECONDS });
  }
}

async function deleteEntry(nodeId, env) {
  if (env.DISCOVERY_KV) {
    await env.DISCOVERY_KV.delete(nodeId);
  }
}

async function listEntries(env) {
  if (env.DISCOVERY_KV) {
    const list = await env.DISCOVERY_KV.list();
    const entries = [];
    for (const key of list.keys) {
      // Skip pending connection keys (they use a different prefix)
      if (key.name.startsWith('pending-')) continue;
      const data = await env.DISCOVERY_KV.get(key.name);
      if (!data) continue;
      try {
        entries.push(JSON.parse(data));
      } catch {
        continue;
      }
    }
    return entries;
  }
  return [];
}

async function handleRegister(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId || !data.publicKey || !data.endpoint) {
    return jsonResponse({ error: 'Missing required fields' }, 400);
  }

  const now = Date.now();
  const existing = await getEntry(data.nodeId, env);
  const entry = {
    nodeId: data.nodeId,
    publicKey: data.publicKey,
    endpoint: data.endpoint,
    displayName: data.displayName || 'Zoop Node',
    region: data.region || 'global',
    isPublic: data.isPublic !== false,
    updatedAt: now,
    createdAt: existing?.createdAt || now,
  };

  await putEntry(data.nodeId, entry, env);
  return jsonResponse({ success: true, entry });
}

async function handleDiscover(request, env) {
  const url = new URL(request.url);
  const region = url.searchParams.get('region') || 'global';
  const entries = await listEntries(env);
  const results = entries.filter((entry) => entry.region === region && entry.isPublic);
  return jsonResponse({ success: true, peers: results });
}

async function handleHeartbeat(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId) {
    return jsonResponse({ error: 'Missing nodeId' }, 400);
  }

  const now = Date.now();
  const entry = await getEntry(data.nodeId, env);
  if (!entry) {
    // Node not registered yet — if they provide enough info, auto-register
    if (data.publicKey && data.endpoint) {
      const newEntry = {
        nodeId: data.nodeId,
        publicKey: data.publicKey,
        endpoint: data.endpoint,
        displayName: data.displayName || 'Zoop Node',
        region: data.region || 'global',
        isPublic: data.isPublic !== false,
        updatedAt: now,
        createdAt: now,
      };
      await putEntry(data.nodeId, newEntry, env);
      return jsonResponse({ success: true, entry: newEntry });
    }
    return jsonResponse({ error: 'Node not registered' }, 404);
  }

  // Update any fields sent with the heartbeat
  if (data.publicKey !== undefined) entry.publicKey = data.publicKey;
  if (data.endpoint !== undefined) entry.endpoint = data.endpoint;
  if (data.displayName !== undefined) entry.displayName = data.displayName;
  if (data.region !== undefined) entry.region = data.region;
  if (data.isPublic !== undefined) entry.isPublic = data.isPublic;
  entry.updatedAt = now;

  await putEntry(data.nodeId, entry, env);
  return jsonResponse({ success: true, entry });
}

async function handleUnregister(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId) {
    return jsonResponse({ error: 'Missing nodeId' }, 400);
  }

  await deleteEntry(data.nodeId, env);
  return jsonResponse({ success: true });
}

async function handleConnectRequest(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId || !data.targetNodeId || !data.clientPubKey) {
    return jsonResponse({ error: 'Missing required fields (nodeId, targetNodeId, clientPubKey)' }, 400);
  }

  const pendingKey = `pending-${data.targetNodeId}`;
  const existing = await getPendingPeers(pendingKey, env);

  // Check for duplicate (same nodeId already has a pending request)
  const dup = existing.find(p => p.nodeId === data.nodeId);
  if (dup) {
    dup.clientPubKey = data.clientPubKey;
    dup.clientIP = data.clientIP || '10.200.0.2/32';
    dup.updatedAt = Date.now();
    await putPendingPeers(pendingKey, existing, env);
    return jsonResponse({ success: true, pending: dup });
  }

  const pending = {
    nodeId: data.nodeId,
    targetNodeId: data.targetNodeId,
    clientPubKey: data.clientPubKey,
    clientIP: data.clientIP || '10.200.0.2/32',
    updatedAt: Date.now(),
  };
  existing.push(pending);
  await putPendingPeers(pendingKey, existing, env);
  return jsonResponse({ success: true, pending });
}

async function handleCancelConnect(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId || !data.targetNodeId) {
    return jsonResponse({ error: 'Missing required fields (nodeId, targetNodeId)' }, 400);
  }

  const pendingKey = `pending-${data.targetNodeId}`;
  const existing = await getPendingPeers(pendingKey, env);
  const filtered = existing.filter(p => p.nodeId !== data.nodeId);
  if (filtered.length === 0) {
    await deleteEntry(pendingKey, env);
  } else {
    await putPendingPeers(pendingKey, filtered, env);
  }
  return jsonResponse({ success: true });
}

async function handlePendingPeers(request, env) {
  const url = new URL(request.url);
  const nodeId = url.searchParams.get('nodeId');
  if (!nodeId) {
    return jsonResponse({ error: 'Missing nodeId query param' }, 400);
  }

  const pendingKey = `pending-${nodeId}`;
  const entries = await getPendingPeers(pendingKey, env);
  return jsonResponse({ success: true, peers: entries });
}

async function getPendingPeers(key, env) {
  if (env.DISCOVERY_KV) {
    const data = await env.DISCOVERY_KV.get(key);
    if (!data) return [];
    try {
      return JSON.parse(data);
    } catch {
      return [];
    }
  }
  return [];
}

async function putPendingPeers(key, entries, env) {
  if (env.DISCOVERY_KV) {
    // Pending connections don't expire via TTL — they stay until processed or cancelled
    await env.DISCOVERY_KV.put(key, JSON.stringify(entries));
  }
}

async function handleDownload(request, env) {
  try {
    const response = await fetch(
      'https://api.github.com/repos/Zoop-Network/zooptray/releases/latest',
      {
        headers: {
          Accept: 'application/vnd.github.v3+json',
          'User-Agent': 'Zoop-Download-Helper',
        },
      }
    );

    if (!response.ok) {
      return new Response('Failed to fetch latest release', { status: 502 });
    }

    const release = await response.json();
    if (!release.assets || release.assets.length === 0) {
      return new Response('No release assets found', { status: 404 });
    }

    const exeAsset = release.assets.find((a) => a.name.endsWith('.exe'));
    const zipAsset = release.assets.find((a) => a.name.endsWith('.zip'));
    const asset = exeAsset || zipAsset;

    if (!asset) {
      return new Response('No installer found in release', { status: 404 });
    }

    return new Response(null, {
      status: 302,
      headers: {
        Location: asset.browser_download_url,
      },
    });
  } catch (error) {
    console.error('Download error:', error);
    return new Response('Internal server error', { status: 500 });
  }
}

async function handleDisconnectNotification(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId || !data.targetNodeId || !data.clientPubKey) {
    return jsonResponse({ error: 'Missing required fields (nodeId, targetNodeId, clientPubKey)' }, 400);
  }

  // Store the disconnect notification so the target node can pick it up
  const disconnectKey = `disconnect-${data.targetNodeId}`;
  const existing = await getDisconnectNotifications(disconnectKey, env);
  
  // Check for duplicate
  const dupIndex = existing.findIndex(d => d.nodeId === data.nodeId && d.clientPubKey === data.clientPubKey);
  if (dupIndex >= 0) {
    // Update timestamp
    existing[dupIndex].updatedAt = Date.now();
  } else {
    // Add new notification
    existing.push({
      nodeId: data.nodeId,
      targetNodeId: data.targetNodeId,
      clientPubKey: data.clientPubKey,
      updatedAt: Date.now(),
    });
  }
  
  await putDisconnectNotifications(disconnectKey, existing, env);
  return jsonResponse({ success: true });
}

async function getDisconnectNotifications(key, env) {
  if (env.DISCOVERY_KV) {
    const data = await env.DISCOVERY_KV.get(key);
    if (!data) return [];
    try {
      return JSON.parse(data);
    } catch {
      return [];
    }
  }
  return [];
}

async function putDisconnectNotifications(key, notifications, env) {
  if (env.DISCOVERY_KV) {
    // Disconnect notifications expire after 60 seconds
    await env.DISCOVERY_KV.put(key, JSON.stringify(notifications), { expirationTtl: 60 });
  }
}

async function handleDisconnectNotifications(request, env) {
  const url = new URL(request.url);
  const nodeId = url.searchParams.get('nodeId');
  if (!nodeId) {
    return jsonResponse({ error: 'Missing nodeId query param' }, 400);
  }

  const disconnectKey = `disconnect-${nodeId}`;
  const notifications = await getDisconnectNotifications(disconnectKey, env);
  
  // Clean up old notifications (older than 30 seconds)
  const now = Date.now();
  const recentNotifications = notifications.filter(n => now - n.updatedAt < 30000);
  
  if (recentNotifications.length > 0) {
    // Update with cleaned list
    await putDisconnectNotifications(disconnectKey, recentNotifications, env);
  } else {
    // No recent notifications, clean up the key
    await env.DISCOVERY_KV.delete(disconnectKey);
  }
  
  return jsonResponse({ success: true, disconnects: recentNotifications });
}

async function handleRelayRequest(request, env) {
  const url = new URL(request.url);
  const targetNodeId = url.searchParams.get('target');
  if (!targetNodeId) {
    return jsonResponse({ error: 'Missing target query param' }, 400);
  }

  // Check if we have any relay servers available
  // In a production system, this would query a pool of relay servers
  // For this implementation, we'll use a simple approach
  
  const relayKey = 'relay-servers';
  let relayServers = [];
  
  if (env.DISCOVERY_KV) {
    const data = await env.DISCOVERY_KV.get(relayKey);
    if (data) {
      try {
        relayServers = JSON.parse(data);
      } catch {
        relayServers = [];
      }
    }
  }
  
  // If no relay servers are configured, return empty
  if (relayServers.length === 0) {
    return jsonResponse({ success: true, relay: '' });
  }
  
  // Simple round-robin selection
  const now = Date.now();
  const index = Math.floor(now / 1000) % relayServers.length;
  const selectedRelay = relayServers[index];
  
  return jsonResponse({ success: true, relay: selectedRelay });
}

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
