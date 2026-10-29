// Own one isolated MySQL database for the lifetime of a complete browser test run.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomBytes } from 'node:crypto';
import { spawn, spawnSync } from 'node:child_process';
import { releaseFixedServicePorts } from '../../scripts/release-service-ports.mjs';

const root = fileURLToPath(new URL('../../', import.meta.url));
const privateEnvironment = path.join(root, '.env');
if (fs.existsSync(privateEnvironment)) process.loadEnvFile(privateEnvironment);
const token = randomBytes(12).toString('hex');
const schema = `leafdoc_test_migration_${token}`;
const environment = {
  ...process.env,
  LEAFDOC_MYSQL_DATABASE: schema,
  LEAFDOC_E2E_DATABASE_OWNED: token,
  LEAFDOC_JWT_SECRET: randomBytes(64).toString('base64'),
  LEAFDOC_ADMIN_USERNAME:
    process.env.E2E_ADMIN_USER || process.env.LEAFDOC_ADMIN_USERNAME || 'admin',
  LEAFDOC_ADMIN_PASSWORD:
    process.env.E2E_ADMIN_PASSWORD ||
    process.env.LEAFDOC_ADMIN_PASSWORD ||
    randomBytes(24).toString('base64url'),
};
environment.E2E_ADMIN_USER = environment.LEAFDOC_ADMIN_USERNAME;
environment.E2E_ADMIN_PASSWORD = environment.LEAFDOC_ADMIN_PASSWORD;
for (const key of ['LEAFDOC_MYSQL_HOST', 'LEAFDOC_MYSQL_USER', 'LEAFDOC_MYSQL_PASSWORD']) {
  if (!environment[key]) throw new Error(`Required MySQL test configuration is missing: ${key}`);
}

function query(sql) {
  const result = spawnSync(
    'mysql',
    [
      '--host',
      environment.LEAFDOC_MYSQL_HOST,
      '--port',
      environment.LEAFDOC_MYSQL_PORT || '3306',
      '--user',
      environment.LEAFDOC_MYSQL_USER,
      '--default-character-set=utf8mb4',
      '--batch',
      '--skip-column-names',
      '--execute',
      sql,
    ],
    {
      encoding: 'utf8',
      env: { ...environment, MYSQL_PWD: environment.LEAFDOC_MYSQL_PASSWORD },
      windowsHide: true,
    },
  );
  if (result.error) throw result.error;
  if (result.status !== 0)
    throw new Error(`MySQL test database operation failed: ${result.stderr.trim()}`);
  return result.stdout.trim();
}

let created = false;
let owned = false;
let exitCode = 1;
try {
  query(`CREATE DATABASE \`${schema}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`);
  created = true;
  query(
    `CREATE TABLE \`${schema}\`.test_database_owner(owner_token VARCHAR(24) PRIMARY KEY, package_name VARCHAR(191) NOT NULL)`,
  );
  query(`INSERT INTO \`${schema}\`.test_database_owner VALUES('${token}','browser-e2e')`);
  owned = true;
  releaseFixedServicePorts(['backend']);
  const bash =
    process.platform === 'win32' ? path.join(process.env.ProgramFiles, 'Git/bin/bash.exe') : 'bash';
  const runner = spawn(bash, [path.join(root, 'tests/e2e/run.sh'), ...process.argv.slice(2)], {
    cwd: root,
    env: environment,
    stdio: 'inherit',
    windowsHide: true,
  });
  const forwardSignal = () => runner.kill('SIGTERM');
  process.on('SIGINT', forwardSignal);
  process.on('SIGTERM', forwardSignal);
  try {
    exitCode = await new Promise((resolve, reject) => {
      runner.once('error', reject);
      runner.once('exit', (code) => resolve(code ?? 1));
    });
  } finally {
    process.off('SIGINT', forwardSignal);
    process.off('SIGTERM', forwardSignal);
  }
} finally {
  if (created) {
    if (
      owned &&
      query(`SELECT owner_token FROM \`${schema}\`.test_database_owner LIMIT 1`) !== token
    ) {
      throw new Error('Refusing to remove a MySQL test database whose ownership changed');
    }
    query(`DROP DATABASE \`${schema}\``);
  }
}
process.exitCode = exitCode;
