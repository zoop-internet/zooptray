# Zoop Peer Discovery Worker

This worker is intended to provide lightweight peer discovery for the Zoop One desktop app using Cloudflare Workers, avoiding a self-hosted VPS.

## Goals

- Register online peers with a short-lived token
- Discover nearby peers by node metadata
- Support simple in-memory or KV backed discovery
- Keep the worker lightweight and network-friendly

## Quick start

1. Install Wrangler:
   ```bash
   npm install -g wrangler
   ```
2. Configure `wrangler.toml` with your account ID.
3. Develop locally:
   ```bash
   cd cloudflare-worker
   wrangler dev
   ```
4. Publish:
   ```bash
   wrangler publish
   ```

## API

- `POST /register` - Register node availability
- `GET /discover?region=...` - Discover nearby nodes
- `POST /heartbeat` - Refresh node registration
- `DELETE /register` - Remove node from discovery
