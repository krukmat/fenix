// admin-screenshots Task 3: Puppeteer navigate + screenshot logic
import puppeteer, { type Browser, type Page } from 'puppeteer';
import * as fs from 'fs';
import * as path from 'path';
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
  token: string,
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

    for (const entry of catalog) {
      const route = entry.url(bffBaseUrl, seed, resolved);
      const outputPath = path.join(outputDir, `${entry.name}.png`);
      const capturedAt = new Date().toISOString();

      try {
        // Inject Bearer token into every request this page makes (initial HTML + HTMX fetches)
        await page.setExtraHTTPHeaders({ Authorization: `Bearer ${token}` });

        await page.goto(route, { waitUntil: 'networkidle2', timeout: 15000 });

        // Inject into localStorage so HTMX client uses it for future sub-requests
        await page.evaluate((t: string) => {
          localStorage.setItem('fenix.admin.bearerToken', t);
        }, token);

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
