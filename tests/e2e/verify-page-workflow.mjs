// Exercises the running instance through its editor, metadata, assets and history UI.
// A uniquely named fixture is removed through the public application interfaces.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, expect } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
process.loadEnvFile(path.join(root, '.env'));
const ports = JSON.parse(await fs.readFile(path.join(root, 'internal/core/serviceports/ports.json'), 'utf8'));
const output = path.join(root, 'build/verification');
const stamp = Date.now().toString(36);
const title = `运行验收 ${stamp}`;
const slug = `runtime-verification-${stamp}`;
const firstContent = `# 运行验收\n\n中文持久化验证 ${stamp}，包含 **格式文本**。`;
const secondText = `第二次编辑 ${stamp}`;
const report = { completed: false, checks: [], errors: [], warnings: [], failedResponses: [] };
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL: `https://localhost:${ports.frontend}`, viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
page.setDefaultTimeout(15000);
page.on('pageerror', error => report.errors.push(error.message));
page.on('console', message => {
  if (message.type() === 'error') report.errors.push(message.text());
  if (message.type() === 'warning') report.warnings.push(message.text());
});
page.on('response', response => {
  if (response.status() >= 400) report.failedResponses.push({ url: response.url(), status: response.status() });
});
let fixture;

// Requests share the browser's secure cookies and the same CSRF protection as UI writes.
async function api(url, method = 'GET', data) {
  const cookies = await context.cookies();
  const csrf = cookies.find(cookie => cookie.name.endsWith('leafdoc_csrf'));
  const response = await context.request.fetch(url, { method, data, headers: csrf ? { 'X-CSRF-Token': decodeURIComponent(csrf.value) } : {} });
  expect(response.ok(), `${method} ${url}: ${await response.text()}`).toBe(true);
  return response.status() === 204 ? undefined : response.json();
}

async function toolbar(id, menuId) {
  const direct = page.getByTestId(id);
  await expect(page.locator('.page-viewer__content')).toBeVisible();
  if (id === 'edit-page-button') {
    await direct.click();
    return;
  }
  if (await direct.isVisible()) await direct.click();
  else {
    await page.getByTestId('toolbar-overflow-button').click();
    await page.getByTestId(menuId).click();
  }
}

async function saveEditor() {
  const saved = page.waitForResponse(response => response.url().endsWith(`/api/pages/${fixture.id}`) && response.request().method() === 'PUT');
  await page.getByTestId('save-page-button').click();
  expect((await saved).ok()).toBe(true);
  await expect(page.locator('.editor-title-bar__dirty-indicator')).toHaveCount(0);
  await page.getByTestId('close-editor-button').click();
  await expect(page.locator('.page-viewer__content')).toBeVisible();
}

try {
  await api('/api/auth/login', 'POST', { identifier: process.env.LEAFDOC_ADMIN_USERNAME, password: process.env.LEAFDOC_ADMIN_PASSWORD });
  fixture = await api('/api/pages', 'POST', { parentId: null, title, slug, kind: 'page' });
  await page.goto(`/${slug}`);
  await toolbar('edit-page-button', 'edit-page-menu-item');
  await page.locator('.cm-content[contenteditable="true"]').fill(firstContent);
  await page.locator('.page-frontmatter-panel__trigger').click();
  await page.getByTestId('page-frontmatter-tag-input').fill('运行验收标签');
  await page.getByTestId('page-frontmatter-tag-input').press('Enter');
  await page.getByTestId('page-frontmatter-add-field').click();
  await page.getByTestId('page-frontmatter-field-key-0').fill('验收状态');
  await page.getByTestId('page-frontmatter-field-value-0').fill('已验证');
  await page.getByTestId('open-asset-manager-button').click();
  await page.locator('input[type="file"]').setInputFiles(path.join(root, 'tests/e2e/assets/upload-test.png'));
  const asset = page.getByTestId('asset-item').filter({ hasText: 'upload-test.png' });
  await expect(asset).toBeVisible();
  await asset.dblclick();
  if (await page.getByRole('dialog').isVisible()) await page.keyboard.press('Escape');
  await saveEditor();
  await expect(page.locator('.page-viewer__content')).toContainText(stamp);
  const image = page.locator('.page-viewer__content img').first();
  await expect(image).toBeVisible();
  await expect.poll(() => image.evaluate(element => element.complete && element.naturalWidth > 0)).toBe(true);
  const savedPage = await api(`/api/pages/${fixture.id}`);
  expect(savedPage.tags).toContain('运行验收标签');
  expect(savedPage.properties['验收状态']).toBe('已验证');
  report.checks.push('真实编辑器保存中文格式文本、标签、属性，上传并插入图片，重读数据与实际图片显示一致');
  await page.screenshot({ path: path.join(output, 'page-workflow-saved.png'), fullPage: true });

  const row = page.getByTestId(`tree-node-${fixture.id}`);
  await row.getByTestId(`tree-node-link-${fixture.id}`).hover();
  await row.getByTestId('tree-view-action-button-open-more-actions').click();
  await page.getByTestId('tree-view-action-button-favorite').click();
  const favorite = page.getByTestId(`favorite-toggle-${fixture.id}`).first();
  await expect(favorite).toHaveAttribute('aria-pressed', 'true');
  await page.reload();
  await expect(page.getByTestId(`favorite-toggle-${fixture.id}`).first()).toHaveAttribute('aria-pressed', 'true');
  await page.getByTestId('sidebar-search-tab-button').click();
  await page.getByTestId('search-input').fill(stamp);
  await expect(page.getByTestId(`search-result-card-${fixture.id}`)).toBeVisible();
  await page.getByTestId(`search-result-card-${fixture.id}`).click();
  report.checks.push('通过收藏按钮保存并刷新确认，通过实际搜索找到中文页面并打开');

  await toolbar('edit-page-button', 'edit-page-menu-item');
  await page.locator('.cm-content[contenteditable="true"]').click();
  await page.keyboard.press('Control+End');
  await page.keyboard.insertText(`\n\n${secondText}`);
  await saveEditor();
  await expect(page.locator('.page-viewer__content')).toContainText(secondText);
  await toolbar('page-history-button', 'page-history-menu-item');
  const revisions = page.locator('button[data-testid^="history-sidebar-revision-"]');
  await expect(revisions.nth(1)).toBeVisible();
  await revisions.nth(1).click();
  await page.getByTestId('page-history-page-restore').click();
  const restored = page.waitForResponse(response => response.request().method() === 'POST' && response.url().includes(`/api/pages/${fixture.id}/revisions/`) && response.url().endsWith('/restore'));
  await page.getByTestId('restore-revision-dialog-button-confirm').click();
  const restoredResponse = await restored;
  expect(restoredResponse.ok()).toBe(true);
  expect((await restoredResponse.json()).content).not.toContain(secondText);
  await expect(page.getByTestId('restore-revision-dialog-button-confirm')).toHaveCount(0);
  await page.goto(`/${slug}`);
  await expect(page.locator('.page-viewer__content')).not.toContainText(secondText);
  await expect(page.locator('.page-viewer__content')).toContainText(stamp);
  report.checks.push('实际进行第二次编辑，进入历史并恢复前一版本，正文回到首次保存内容');
  await toolbar('delete-page-button', 'delete-page-menu-item');
  await expect(page.getByTestId('delete-page-dialog-backlinks-loading')).toHaveCount(0);
  const deleted = page.waitForResponse(response => response.request().method() === 'DELETE' && response.url().includes(`/api/pages/${fixture.id}`));
  await page.getByTestId('delete-page-dialog-button-confirm').click();
  expect((await deleted).ok()).toBe(true);
  await expect(page.getByTestId('delete-page-dialog-button-confirm')).toHaveCount(0);
  await expect(page).not.toHaveURL(new RegExp(slug));
  const tree = await api('/api/tree');
  expect(tree.children).toHaveLength(5);
  expect(tree.children.some(node => node.id === fixture.id)).toBe(false);
  fixture = undefined;
  expect(report.errors).toEqual([]);
  expect(report.warnings).toEqual([]);
  expect(report.failedResponses).toEqual([]);
  report.checks.push('通过删除确认清理临时页面，最终仅保留五篇默认文档');
  report.completed = true;
} catch (error) {
  report.failure = error.stack || error.message;
  report.failedURL = page.url();
  await page.screenshot({ path: path.join(output, 'page-workflow-failure.png'), fullPage: true });
  process.exitCode = 1;
} finally {
  if (fixture) {
    try {
      const current = await api(`/api/pages/${fixture.id}`);
      await api(`/api/pages/${fixture.id}?version=${encodeURIComponent(current.version)}`, 'DELETE');
      report.fixtureCleaned = true;
    } catch (error) {
      report.errors.push(`临时页面清理失败: ${error.message}`);
    }
  }
  await context.close();
  await browser.close();
  if (report.errors.length || report.warnings.length || report.failedResponses.length) {
    report.completed = false;
    process.exitCode = 1;
  }
  await fs.writeFile(path.join(output, 'page-workflow-report.json'), JSON.stringify(report, null, 2) + '\n', 'utf8');
}
console.log(JSON.stringify(report, null, 2));
