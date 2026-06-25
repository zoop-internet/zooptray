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

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
