// admin-screenshots Task 3: Puppeteer navigate + screenshot logic
import puppeteer, { type Browser, type Page } from 'puppeteer';
import * as fs from 'fs';
import * as path from 'path';
import axios from 'axios';
import type { SeederOutput } from '../snapshots/types';
import { catalog, type ResolvedIds } from './catalog';

export type ShooterResult = {
  name: string;
  route: string;
  outputPath: string;
  capturedAt: string;
  ok: boolean;
  error?: string;
};

export async function runShooter(
  bffBaseUrl: string,
  seed: SeederOutput,
  resolved: ResolvedIds,
  outputDir: string,
): Promise<ShooterResult[]> {
  fs.mkdirSync(outputDir, { recursive: true });

  const browser: Browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const results: ShooterResult[] = [];

  try {
    const page: Page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 900 });
    let sessionEstablished = false;

    for (const entry of catalog) {
      const route = entry.url(bffBaseUrl, seed, resolved);
      const outputPath = path.join(outputDir, `${entry.name}.png`);
      const capturedAt = new Date().toISOString();

      try {
        if (entry.requiresSession && !sessionEstablished) {
          await loginAsAdmin(page, bffBaseUrl, seed.credentials.email, seed.credentials.password);
          sessionEstablished = true;
        }

        await page.goto(route, { waitUntil: 'networkidle2', timeout: 15000 });

        // Allow HTMX fragments to settle after networkidle2
        await new Promise((resolve) => setTimeout(resolve, 500));

        await page.screenshot({ path: outputPath, fullPage: true });

        process.stdout.write(`[admin-screenshots] ✓ ${entry.name}\n`);
        results.push({ name: entry.name, route, outputPath, capturedAt, ok: true });
      } catch (err: unknown) {
        const error = err instanceof Error ? err.message : String(err);
        process.stderr.write(`[admin-screenshots] ✗ ${entry.name}: ${error}\n`);
        results.push({ name: entry.name, route, outputPath, capturedAt, ok: false, error });
      }
    }
  } finally {
    await browser.close();
  }

  return results;
}

async function loginAsAdmin(page: Page, bffBaseUrl: string, email: string, password: string): Promise<void> {
  // POST from Node (not the browser) so we can read the Set-Cookie response header directly.
  // HttpOnly cookies are invisible to page.evaluate JS, so we inject the cookie via page.setCookie().
  const loginRes = await axios.post(
    `${bffBaseUrl}/bff/admin/login`,
    `email=${encodeURIComponent(email)}&password=${encodeURIComponent(password)}`,
    {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      maxRedirects: 0,
      validateStatus: (s) => s === 302,
    },
  );

  const setCookieHeader = loginRes.headers['set-cookie'];
  const cookieArray = Array.isArray(setCookieHeader) ? setCookieHeader : setCookieHeader ? [setCookieHeader] : [];
  const sessionRaw = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid='));

  if (!sessionRaw) {
    throw new Error('Admin login did not produce fenix.admin.sid session cookie');
  }

  // Extract value (everything between = and first ;)
  const sessionValue = sessionRaw.split('=')[1]?.split(';')[0] ?? '';

  const url = new URL(bffBaseUrl);
  await page.setCookie({
    name: 'fenix.admin.sid',
    value: sessionValue,
    domain: url.hostname,
    path: '/',
    httpOnly: true,
    sameSite: 'Lax',
  });
}
