// Reverse-proxy tests use fixed shared ports and a disposable private MySQL service.
import { readFileSync } from 'node:fs';
import { randomBytes } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const directory = path.dirname(fileURLToPath(import.meta.url));
const allocation = JSON.parse(readFileSync(path.join(directory, '../../internal/core/serviceports/ports.json'), 'utf8'));
const runID = randomBytes(12).toString('hex');
const environment = {
  ...process.env,
  COMPOSE_PROJECT_NAME: `leafdoc-proxy-${runID}`,
  COMPOSE_ENV_FILES: '',
  COMPOSE_DISABLE_ENV_FILE: 'true',
  E2E_PROXY_RUN_ID: runID,
  E2E_BACKEND_PORT: String(allocation.backend),
  E2E_PROXY_PORT: String(allocation.proxyTest),
  E2E_AUTOCREATE_PROXY_PORT: String(allocation.proxyAutoCreateTest),
  E2E_MYSQL_PORT: String(allocation.proxyTestMySQL),
  E2E_MYSQL_PASSWORD: randomBytes(32).toString('hex'),
  E2E_JWT_SECRET: randomBytes(32).toString('hex'),
  E2E_DATABASE: `leafdoc_proxy_test_${runID}_default`,
  E2E_AUTOCREATE_DATABASE: `leafdoc_proxy_test_${runID}_autocreate`,
  E2E_PROXY_URL: `http://127.0.0.1:${allocation.proxyTest}`,
  E2E_AUTOCREATE_PROXY_URL: `http://127.0.0.1:${allocation.proxyAutoCreateTest}`,
};
for (const service of ['backend', 'proxyTest', 'proxyAutoCreateTest', 'proxyTestMySQL']) {
  if (!Number.isInteger(allocation[service]) || allocation[service] < 1024 || allocation[service] > 65535) {
    throw new Error(`Invalid fixed port allocation: ${service}`);
  }
}
function execute(command, args, options = {}) {
  const result = spawnSync(command, args, { cwd: directory, env: environment, stdio: 'inherit', ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} exited with status ${result.status}`);
}
// Config validation does not create containers, publish ports or connect to any database.
execute('docker', ['compose', 'config', '--quiet']);
if (process.argv.includes('--check')) {
  const result = spawnSync('docker', ['compose', 'config', '--format', 'json'], { cwd: directory, env: environment, encoding: 'utf8' });
  if (result.status !== 0) throw new Error('Compose configuration validation failed');
  const config = JSON.parse(result.stdout);
  for (const service of ['leafdoc', 'leafdoc-autocreate']) {
    const env = config.services[service].environment;
    if (env.LEAFDOC_MYSQL_HOST !== 'mysql' || !env.LEAFDOC_MYSQL_DATABASE.startsWith(`leafdoc_proxy_test_${runID}_`)) throw new Error('Unsafe test database configuration');
    if (config.services[service].ports?.length) throw new Error('Backend test ports must remain private');
  }
  if (config.services.mysql.ports?.length) throw new Error('MySQL test service must remain private');
  const exposed = ['nginx', 'nginx-autocreate'].flatMap(service => config.services[service].ports.map(port => Number(port.published)));
  if (exposed.includes(allocation.frontend)) throw new Error('Test stack must not publish the production frontend port');
  console.log('Compose configuration valid; MySQL schemas and container network isolated; no containers started.');
} else {
  let failed = false;
  const cleanup = () => execute('docker', ['compose', 'down', '--volumes', '--remove-orphans']);
  try {
    execute('docker', ['compose', 'up', '-d', '--build', '--wait']);
    execute('go', ['test', '-v', '-timeout', '60s', './...']);
  } catch (error) {
    failed = true;
    console.error(error.message);
  } finally {
    try { cleanup(); } catch (error) { failed = true; console.error(`Test container cleanup failed: ${error.message}`); }
  }
  if (failed) process.exitCode = 1;
}
