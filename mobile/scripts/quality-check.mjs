#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';

const ROOT = process.cwd();
const MOBILE_SRC = path.join(ROOT, 'src');
const ALLOWED_URL_CONFIG_FILES = new Set([
  path.join('src', 'services', 'api.ts'),
]);

const violations = [];

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (['node_modules', 'dist', 'coverage'].includes(entry.name)) continue;
      files.push(...walk(fullPath));
    } else if (entry.isFile()) {
      files.push(fullPath);
    }
  }

  return files;
}

function checkFile(filePath) {
  const rel = path.relative(ROOT, filePath);
  const content = fs.readFileSync(filePath, 'utf8');
  const normalizedRel = rel.split(path.sep).join('/');

  // Enforce BFF-only calls (no direct :8080 backend calls)
  if (content.match(/https?:\/\/[^\s'"`]*:8080/g)) {
    violations.push(`${rel}: direct backend call to :8080 detected (must use BFF)`);
  }

  // Detect hardcoded URLs outside config/env files
  const hardcodedUrlMatches = content.match(/https?:\/\/[^\s'"`]+/g) || [];
  if (hardcodedUrlMatches.length > 0 && !ALLOWED_URL_CONFIG_FILES.has(rel)) {
    violations.push(`${rel}: hardcoded URL detected (must come from config/env)`);
  }

  // React Query query keys in CRM hooks should include workspace isolation
  if (normalizedRel === 'src/hooks/useCRM.ts' && content.includes('useQuery({')) {
    const queryKeyExprs = [...content.matchAll(/queryKey\s*:\s*([^,\n]+)/g)].map((m) => m[1] ?? '');
    queryKeyExprs.forEach((expr) => {
      if (!expr.includes('workspaceId')) {
        violations.push(
          `${rel}: queryKey without workspace isolation detected (${expr.trim()})`
        );
      }
    });
  }

  // Restrict app-layer imports: app routes/layouts should not access API service directly
  if (normalizedRel.startsWith('app/') && content.match(/from\s+['"][^'"]*services\/api['"]/)) {
    violations.push(`${rel}: app layer must not import services/api directly (use hooks/store layer)`);
  }

  // Avoid unstable list keys from Math.random
  if (content.includes('keyExtractor') && content.includes('Math.random()')) {
    violations.push(`${rel}: keyExtractor uses Math.random() (unstable keys)`);
  }

  // Avoid unstable list keys from index usage
  if (content.match(/key\s*=\s*\{\s*index\s*\}/)) {
    violations.push(`${rel}: list item key uses index (unstable keys)`);
  }
  if (
    content.match(
      /keyExtractor\s*=\s*\{\s*\([^)]*\bindex\b[^)]*\)\s*=>\s*(index(\.toString\(\))?|`\$\{index\}`)\s*\}/
    )
  ) {
    violations.push(`${rel}: keyExtractor returns index (unstable keys)`);
  }

  // Forbid explicit any in source code
  if (content.match(/:\s*any\b/g)) {
    violations.push(`${rel}: explicit any detected`);
  }
}

function main() {
  if (!fs.existsSync(MOBILE_SRC)) {
    console.error('quality-check: src directory not found');
    process.exit(1);
  }

  const files = walk(MOBILE_SRC).filter((f) => /\.(ts|tsx)$/.test(f));
  files.forEach(checkFile);

  if (violations.length > 0) {
    console.error('❌ Mobile quality architecture checks failed:\n');
    violations.forEach((v) => console.error(`- ${v}`));
    process.exit(1);
  }

  console.log('✅ Mobile quality architecture checks passed');
}

main();
