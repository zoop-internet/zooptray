export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const { pathname, searchParams } = url;
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

    return new Response(JSON.stringify({ error: 'Not Found' }), {
      status: 404,
      headers: { 'Content-Type': 'application/json' },
    });
  },
};

const REGISTRY = new Map();

async function handleRegister(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId || !data.publicKey || !data.endpoint) {
    return jsonResponse({ error: 'Missing required fields' }, 400);
  }

  const now = Date.now();
  const entry = {
    nodeId: data.nodeId,
    publicKey: data.publicKey,
    endpoint: data.endpoint,
    displayName: data.displayName || 'Zoop Node',
    region: data.region || 'global',
    updatedAt: now,
    createdAt: REGISTRY.has(data.nodeId) ? REGISTRY.get(data.nodeId).createdAt : now,
  };

  REGISTRY.set(data.nodeId, entry);
  return jsonResponse({ success: true, entry });
}

async function handleDiscover(request, env) {
  const url = new URL(request.url);
  const region = url.searchParams.get('region') || 'global';
  const results = Array.from(REGISTRY.values()).filter((entry) => entry.region === region);
  return jsonResponse({ success: true, peers: results });
}

async function handleHeartbeat(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId) {
    return jsonResponse({ error: 'Missing nodeId' }, 400);
  }

  const entry = REGISTRY.get(data.nodeId);
  if (!entry) {
    return jsonResponse({ error: 'Node not registered' }, 404);
  }

  entry.updatedAt = Date.now();
  REGISTRY.set(data.nodeId, entry);
  return jsonResponse({ success: true, entry });
}

async function handleUnregister(request, env) {
  const data = await request.json().catch(() => null);
  if (!data || !data.nodeId) {
    return jsonResponse({ error: 'Missing nodeId' }, 400);
  }

  REGISTRY.delete(data.nodeId);
  return jsonResponse({ success: true });
}

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
