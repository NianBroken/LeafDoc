// Captures the final accepted document state without repeating destructive restore checks.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, expect } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
process.loadEnvFile(path.join(root, '.env'));
const allocation = JSON.parse(
  await fs.readFile(path.join(root, 'internal/core/serviceports/ports.json'), 'utf8'),
);
const output = path.join(root, 'build/verification');
const reportPath = path.join(output, 'settings-report.json');
const report = JSON.parse(await fs.readFile(reportPath, 'utf8'));
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  baseURL: process.env.E2E_BASE_URL || `https://localhost:${allocation.frontend}`,
  acceptDownloads: true,
});
const page = await context.newPage();
const errors = [];
let accountCleanup;
page.on('pageerror', (error) => errors.push(error.message));
page.on('console', (message) => {
  if (['error', 'warning'].includes(message.type())) errors.push(message.text());
});
try {
  await page.goto('/login');
  await page
    .getByTestId('login-identifier')
    .fill(process.env.E2E_ADMIN_USER || process.env.LEAFDOC_ADMIN_USERNAME);
  await page
    .getByTestId('login-password')
    .fill(process.env.E2E_ADMIN_PASSWORD || process.env.LEAFDOC_ADMIN_PASSWORD);
  await page.getByTestId('login-submit').click();
  await expect(page.getByTestId('user-menu-avatar')).toBeVisible();
  if (process.env.E2E_REMOVE_QWERTYUI === '1') {
    await page.goto('/settings/users');
    await expect(
      page.locator('tbody tr', { has: page.getByText('admin', { exact: true }) }),
    ).toBeVisible();
    const target = page.locator('tbody tr', { has: page.getByText('qwertyui', { exact: true }) });
    await expect(target).toBeVisible();
    await target.getByRole('button', { name: 'qwertyui 的更多操作', exact: true }).click();
    await page.getByRole('menuitem', { name: '删除', exact: true }).click();
    const deleted = page.waitForResponse(
      (response) =>
        response.request().method() === 'DELETE' && response.url().includes('/api/users/'),
    );
    await page.getByTestId('delete-user-dialog-button-confirm').click();
    const response = await deleted;
    expect(response.ok()).toBe(true);
    await expect(target).toHaveCount(0);
    await page.reload();
    await expect(page.locator('tbody tr td:first-child')).toHaveText(['admin']);
    await page.screenshot({
      path: path.join(output, 'account-cleanup-admin-only.png'),
      fullPage: true,
    });
    accountCleanup = {
      authorization: '用户明确回复删除qwertyui，只保留admin',
      deletedUsername: 'qwertyui',
      deleteStatus: response.status(),
      remainingUsernamesAfterReload: ['admin'],
    };
  }
  const existingResponse = await context.request.get('/api/admin/snapshot');
  expect(existingResponse.ok()).toBe(true);
  const existing = (await existingResponse.json()).snapshots ?? [];
  const beforeIds = new Set(existing.map((entry) => entry.id));
  await page.goto('/settings/snapshots');
  const rows = page.locator('.snapshot__table tbody tr');
  await expect(rows).toHaveCount(existing.length);
  await page.getByRole('button', { name: '立即创建备份', exact: true }).click();
  const newIds = async () =>
    (await rows.locator('td:first-child').allTextContents()).filter((id) => !beforeIds.has(id));
  await expect.poll(newIds, { timeout: 60000 }).toHaveLength(1);
  const finalBackupId = (await newIds())[0];
  const row = rows.filter({ has: page.getByText(finalBackupId, { exact: true }) });
  const downloaded = page.waitForEvent('download');
  await row.getByRole('link', { name: '下载', exact: true }).click();
  const download = await downloaded;
  await download.saveAs(path.join(output, 'verified-backup.zip'));
  expect(await download.failure()).toBeNull();
  await context.close();
  expect(errors).toEqual([]);
  if (accountCleanup) {
    await fs.writeFile(
      path.join(output, 'account-cleanup-report.json'),
      JSON.stringify({ ...accountCleanup, finalBackupId, completed: true, errors }, null, 2) + '\n',
      'utf8',
    );
    report.accountCleanup = accountCleanup;
    report.checks.push('按用户明确授权通过界面删除qwertyui，重载确认仅admin');
  }
  report.finalBackupId = finalBackupId;
  report.checks.push('通过界面创建并下载包含最终五篇正文的完整备份');
  await fs.writeFile(reportPath, JSON.stringify(report, null, 2) + '\n', 'utf8');
  console.log(JSON.stringify({ finalBackupId, errors }));
} finally {
  await browser.close();
}
