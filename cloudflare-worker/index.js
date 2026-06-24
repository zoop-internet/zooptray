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

  return REGISTRY.get(nodeId) || null;
}

async function putEntry(nodeId, entry, env) {
  if (env.DISCOVERY_KV) {
    await env.DISCOVERY_KV.put(nodeId, JSON.stringify(entry), { expirationTtl: DEFAULT_TTL_SECONDS });
    return;
  }
  REGISTRY.set(nodeId, entry);
}

async function deleteEntry(nodeId, env) {
  if (env.DISCOVERY_KV) {
    await env.DISCOVERY_KV.delete(nodeId);
    return;
  }
  REGISTRY.delete(nodeId);
}

async function listEntries(env) {
  if (env.DISCOVERY_KV) {
    const list = await env.DISCOVERY_KV.list();
    const entries = [];
    for (const key of list.keys) {
      const data = await env.DISCOVERY_KV.get(key.name);
      if (!data) {
        continue;
      }
      try {
        entries.push(JSON.parse(data));
      } catch {
        continue;
      }
    }
    return entries;
  }

  return Array.from(REGISTRY.values());
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

  const entry = await getEntry(data.nodeId, env);
  if (!entry) {
    return jsonResponse({ error: 'Node not registered' }, 404);
  }

  entry.updatedAt = Date.now();
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

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
