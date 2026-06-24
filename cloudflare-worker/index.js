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

async function handleDownload(request, env) {
  try {
    // Fetch latest release from GitHub API
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

    // Prefer .exe installer, fallback to .zip
    const exeAsset = release.assets.find((a) => a.name.endsWith('.exe'));
    const zipAsset = release.assets.find((a) => a.name.endsWith('.zip'));
    const asset = exeAsset || zipAsset;

    if (!asset) {
      return new Response('No installer found in release', { status: 404 });
    }

    // Redirect to the download URL
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
