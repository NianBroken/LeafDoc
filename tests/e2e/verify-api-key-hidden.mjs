// Confirms that the deployed feature flag hides navigation and guards direct URLs.
// The check reads existing data and leaves the application services running.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, expect } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
process.loadEnvFile(path.join(root, '.env'));
const ports = JSON.parse(await fs.readFile(path.join(root, 'internal/core/serviceports/ports.json'), 'utf8'));
const output = path.join(root, 'build/verification');
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL: `https://localhost:${ports.frontend}`, viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
const report = { completed: false, errors: [], warnings: [], failedResponses: [], screenshots: [] };
page.on('pageerror', error => report.errors.push(error.message));
page.on('console', message => {
  if (message.type() === 'error') report.errors.push(message.text());
  if (message.type() === 'warning') report.warnings.push(message.text());
});
page.on('response', response => {
  if (response.status() >= 400) report.failedResponses.push({ url: response.url(), status: response.status() });
});
try {
  await page.goto('/login');
  await page.getByTestId('login-identifier').fill(process.env.LEAFDOC_ADMIN_USERNAME);
  await page.getByTestId('login-password').fill(process.env.LEAFDOC_ADMIN_PASSWORD);
  await page.getByTestId('login-submit').click();
  await expect(page.getByTestId('user-menu-avatar')).toBeVisible();
  const config = await page.evaluate(() => fetch('/api/config').then(response => response.json()));
  expect(config.enableApiKeyManagement).toBe(false);
  report.apiKeyManagementEnabled = config.enableApiKeyManagement;
  await page.goto('/settings/branding');
  await expect(page.getByRole('heading', { name: '外观设置', exact: true })).toBeVisible();
  await expect(page.getByTestId('settings-nav-item-api-keys')).toHaveCount(0);
  report.visibleSettingsItems = await page.locator('[data-testid^="settings-nav-item-"]').allTextContents();
  await page.screenshot({ path: path.join(output, 'api-key-hidden-desktop.png'), fullPage: true });
  report.screenshots.push('api-key-hidden-desktop.png');
  await page.goto('/settings/api-keys');
  await expect(page).toHaveURL(/\/settings\/preferences$/);
  report.directApiKeyURLRedirectedTo = new URL(page.url()).pathname;
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/settings/branding');
  const toggle = page.getByTestId('sidebar-toggle-button');
  if (await toggle.getAttribute('aria-expanded') !== 'true') await toggle.click();
  await expect(page.getByTestId('settings-nav')).toBeVisible();
  await expect(page.getByTestId('settings-nav-item-api-keys')).toHaveCount(0);
  await page.screenshot({ path: path.join(output, 'api-key-hidden-mobile.png'), fullPage: true });
  report.screenshots.push('api-key-hidden-mobile.png');
  report.completed = true;
} catch (error) {
  report.failure = error.stack || error.message;
  process.exitCode = 1;
} finally {
  await context.close();
  await browser.close();
  if (report.errors.length || report.warnings.length || report.failedResponses.length) {
    report.completed = false;
    process.exitCode = 1;
  }
  await fs.writeFile(path.join(output, 'api-key-hidden-report.json'), JSON.stringify(report, null, 2) + '\n', 'utf8');
}
console.log(JSON.stringify(report, null, 2));
