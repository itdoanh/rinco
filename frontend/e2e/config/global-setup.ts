/**
 * WS-T: global setup — waits for all 4 dev servers to be reachable,
 * captures initial state into logs/wst-setup.txt, and writes a
 * `playwright/.auth/state.json` placeholder for future sessions.
 *
 * In a future iteration this file should poll the actual backend
 * (auth:8081 /healthz) and seed admin email/password from the seed
 * files; for now the dev servers are the source of truth and they
 * were started manually before `playwright test`.
 */
import { request } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';

const SERVERS = [
  { name: 'landing', url: 'http://localhost:3000' },
  { name: 'admin-portal', url: 'http://localhost:3001' },
  { name: 'tenant-site', url: 'http://localhost:3002' },
  { name: 'meeting-ui', url: 'http://localhost:3003' },
];

const LOG_DIR = resolve(__dirname, '..', '..', '..', 'logs');
mkdirSync(LOG_DIR, { recursive: true });

export default async function globalSetup(): Promise<void> {
  const lines: string[] = [];
  lines.push(`[WS-T global-setup] ${new Date().toISOString()}`);

  for (const srv of SERVERS) {
    try {
      const ctx = await request.newContext({ baseURL: srv.url, timeout: 30_000 });
      const res = await ctx.get('/').catch((e) => ({ status: () => `ERR ${e.message}` }) as any);
      lines.push(`  ${srv.name} ${srv.url} -> status=${res?.status?.() ?? '??'}`);
      await ctx.dispose();
    } catch (e) {
      lines.push(`  ${srv.name} ${srv.url} -> ERR ${(e as Error).message}`);
    }
  }

  writeFileSync(
    resolve(LOG_DIR, 'wst-setup.txt'),
    lines.join('\n') + '\n',
    { flag: 'a' }
  );

  // Optional: write a placeholder auth file so future specs can use it.
  const authDir = resolve(__dirname, '..', '.auth');
  mkdirSync(authDir, { recursive: true });
  writeFileSync(
    resolve(authDir, 'state.json'),
    JSON.stringify({ note: 'No real backend seeded; login flow uses direct form fill.' }, null, 2)
  );
}