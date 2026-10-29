import fs from 'node:fs';
import path from 'node:path';

// Browser helpers share the application's authoritative listener allocation.
const ports = JSON.parse(
  fs.readFileSync(
    path.resolve(__dirname, '../../../internal/core/serviceports/ports.json'),
    'utf8',
  ),
) as Record<string, number>;

export const e2eBaseUrl = process.env.E2E_BASE_URL || `https://localhost:${ports.frontend}`;
