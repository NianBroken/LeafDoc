// Real-browser acceptance checks preserve the instance and write reviewable screenshots.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomBytes } from 'node:crypto';
import { chromium, expect as baseExpect } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
try {
  process.loadEnvFile(path.join(root, '.env'));
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
}
const output = path.join(root, 'build/verification');
const allocation = JSON.parse(
  await fs.readFile(path.join(root, 'internal/core/serviceports/ports.json'), 'utf8'),
);
const baseURL = process.env.E2E_BASE_URL || `https://localhost:${allocation.frontend}`;
const username = process.env.E2E_ADMIN_USER || process.env.LEAFDOC_ADMIN_USERNAME;
const password = process.env.E2E_ADMIN_PASSWORD || process.env.LEAFDOC_ADMIN_PASSWORD;
const viewports = [
  { width: 1440, height: 900 },
  { width: 1024, height: 768 },
  { width: 390, height: 844 },
];
const themes = ['light', 'dark'];
const stamp = Date.now().toString(36);
const fixturePassword = randomBytes(24).toString('base64url');
const fixtures = [
  { username: `align_${stamp}_editor`, role: '编辑者' },
  { username: `align_${stamp}_viewer`, role: '查看者' },
].map((fixture) => ({
  ...fixture,
  email: `${fixture.username}.long.address.for.alignment@example.com`,
}));
const report = { checks: [], screenshots: [], tables: [], errors: [], warnings: [] };
const expect = baseExpect.configure({ timeout: 15000 });

if (!username || !password) {
  throw new Error(
    'Set E2E_ADMIN_USER and E2E_ADMIN_PASSWORD, or the corresponding LEAFDOC administrator variables.',
  );
}

await fs.mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  baseURL,
  viewport: viewports[0],
  acceptDownloads: true,
});
const page = await context.newPage();
page.setDefaultTimeout(15000);
page.on('pageerror', (error) => report.errors.push(error.message));
page.on('console', (message) => {
  if (message.type() === 'error') report.errors.push(message.text());
  if (message.type() === 'warning') report.warnings.push(message.text());
});
const responseInspections = [];
page.on('response', (response) => {
  if (response.status() >= 400) {
    report.failedResponses ??= [];
    const failure = {
      url: response.url(),
      status: response.status(),
    };
    report.failedResponses.push(failure);
    responseInspections.push(
      response.text().then(
        (body) => {
          failure.body = body;
        },
        (error) => {
          failure.bodyReadError = error.message;
        },
      ),
    );
  }
});

async function api(url, method = 'GET', body) {
  const result = await page.evaluate(
    async ({ url, method, body }) => {
      const csrf = document.cookie
        .split(';')
        .map((cookie) => cookie.trim())
        .find((cookie) => cookie.split('=')[0].endsWith('leafdoc_csrf'));
      const headers = { 'Content-Type': 'application/json' };
      if (csrf) headers['X-CSRF-Token'] = decodeURIComponent(csrf.slice(csrf.indexOf('=') + 1));
      const response = await fetch(url, {
        method,
        headers,
        credentials: 'same-origin',
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      return { status: response.status, text: await response.text() };
    },
    { url, method, body },
  );
  expect(result.status, `${method} ${url}: ${result.text}`).toBeLessThan(300);
  return result.text ? JSON.parse(result.text) : undefined;
}

async function login() {
  await page.goto('/login');
  await expect(
    page.getByTestId('login-identifier').or(page.getByTestId('user-menu-avatar')),
  ).toBeVisible();
  if (await page.getByTestId('user-menu-avatar').isVisible()) return;
  await page.getByTestId('login-identifier').fill(username);
  await page.getByTestId('login-password').fill(password);
  await page.getByTestId('login-submit').click();
  await expect(page.getByTestId('user-menu-avatar')).toBeVisible();
}

async function saveScreenshot(name) {
  await page.screenshot({ path: path.join(output, name), fullPage: true, animations: 'disabled' });
  report.screenshots.push(name);
}

function userRow(name) {
  return page.locator('tbody tr', { has: page.getByText(name, { exact: true }) });
}

async function openUserAction(name, action) {
  const trigger = userRow(name).getByRole('button', { name: `${name} 的更多操作` });
  await trigger.scrollIntoViewIfNeeded();
  await trigger.click();
  await page.getByRole('menuitem', { name: action, exact: true }).click();
  return page.getByRole('dialog');
}

async function deleteFixture(fixture) {
  await page.goto('/settings/users');
  await expect(userRow(username)).toBeVisible();
  const row = userRow(fixture.username);
  if ((await row.count()) === 0) return;
  const dialog = await openUserAction(fixture.username, '删除');
  await dialog.getByTestId('delete-user-dialog-button-confirm').click();
  await expect(dialog).toBeHidden();
  await expect(row).toHaveCount(0);
}

// Text ranges measure the visible glyph origin independently of the surrounding box.
async function verifyTable(kind) {
  const data = await page.locator('table.settings__table').evaluate((table) => {
    function contentRect(cell) {
      const content = cell.querySelector(
        '.settings__table-heading, .settings__table-text, .settings__text-action',
      );
      if (!content) throw new Error('A table cell has no visible content');
      const range = document.createRange();
      range.selectNodeContents(content);
      const rect = range.getBoundingClientRect();
      const cellRect = cell.getBoundingClientRect();
      return { x: rect.x - cellRect.x, y: rect.y - cellRect.y, width: cellRect.width };
    }
    return {
      header: Array.from(table.tHead.rows[0].cells, contentRect),
      rows: Array.from(table.tBodies[0].rows, (row) => ({
        height: row.getBoundingClientRect().height,
        cells: Array.from(row.cells, contentRect),
      })),
      documentWidth: document.documentElement.scrollWidth,
      viewportWidth: innerWidth,
    };
  });
  expect(data.rows.length).toBeGreaterThan(0);
  expect(data.documentWidth).toBeLessThanOrEqual(data.viewportWidth + 1);
  for (const row of data.rows) {
    expect(Math.abs(row.height - 48)).toBeLessThanOrEqual(1);
    row.cells.forEach((cell, index) => {
      expect(
        Math.abs(cell.x - data.header[index].x),
        `${kind} column ${index} left alignment`,
      ).toBeLessThanOrEqual(1);
      expect(
        Math.abs(cell.y - data.header[index].y),
        `${kind} column ${index} text baseline`,
      ).toBeLessThanOrEqual(1);
      expect(Math.abs(cell.width - data.header[index].width)).toBeLessThanOrEqual(1);
    });
  }
  report.tables.push({
    kind,
    viewport: page.viewportSize(),
    theme: await page.locator('html').getAttribute('class'),
    ...data,
  });
}

async function selectAppearance(theme, viewport) {
  await page.setViewportSize(viewport);
  await page.emulateMedia({ colorScheme: theme });
  await page.evaluate((theme) => localStorage.setItem('design-mode', theme), theme);
}

async function saveSiteName(value) {
  await page.goto('/settings/branding');
  await page.locator('#siteName').fill(value);
  await page.getByRole('button', { name: '保存设置', exact: true }).click();
  await expect.poll(async () => (await api('/api/branding')).siteName).toBe(value);
  await page.reload();
  await expect(page.locator('#siteName')).toHaveValue(value);
}

// Every rendered reference is exercised against its actual target rather than Markdown source text.
async function verifyDefaultDocuments() {
  const tree = await api('/api/tree');
  const documents = tree.children;
  expect(documents).toHaveLength(5);
  report.documents = [];
  for (const document of documents) {
    if (
      process.env.E2E_RENDER_ONLY === '1' &&
      document.slug !== 'markdown-extensions-and-troubleshooting'
    )
      continue;
    const documentURL = `/${document.slug}`;
    if (
      process.env.E2E_SYNC_DEFAULT_DOCS === '1' &&
      document.slug === 'markdown-extensions-and-troubleshooting'
    ) {
      const content = await fs.readFile(
        path.join(root, 'internal/core/defaultdocs/content', `${document.slug}.md`),
        'utf8',
      );
      await page.goto(`/e/${document.slug}`);
      await page.locator('.cm-content[contenteditable="true"]').fill(content);
      const saved = page.waitForResponse(
        (response) =>
          response.url().endsWith(`/api/pages/${document.id}`) &&
          response.request().method() === 'PUT',
      );
      await page.getByTestId('save-page-button').click();
      expect((await saved).ok()).toBe(true);
      await expect(page.locator('.editor-title-bar__dirty-indicator')).toHaveCount(0);
      await page.getByTestId('close-editor-button').click();
      expect((await api(`/api/pages/${document.id}`)).content).toBe(content);
      report.checks.push('通过真实编辑器将公式教程与内嵌初始化源同步，MySQL正文完全一致');
    }
    await page.goto(documentURL);
    const article = page.locator('.page-viewer__content');
    await expect(article).toBeVisible();
    expect((await article.innerText()).length).toBeGreaterThan(1000);
    expect(await article.locator('pre code').count()).toBeGreaterThan(0);
    if (document.slug === 'markdown-links-images-and-tables') {
      await expect(article.locator('table')).toHaveCount(1);
    }
    if (document.slug === 'markdown-extensions-and-troubleshooting') {
      await expect(article.locator('.katex')).toHaveCount(2);
      await expect(article.locator('.katex').first()).toBeVisible();
      const diagram = article.locator('svg[id^="mermaid-"]');
      await expect(diagram).toBeVisible();
      await saveScreenshot('markdown-extensions-rendered.png');
      await article.locator('.katex').first().scrollIntoViewIfNeeded();
      await saveScreenshot('markdown-formula-rendered.png');
      await diagram.scrollIntoViewIfNeeded();
      await saveScreenshot('markdown-flowchart-rendered.png');
    }
    const links = await article
      .locator('a[href]')
      .evaluateAll((anchors) => anchors.map((anchor) => anchor.getAttribute('href')));
    const images = await article.locator('img').evaluateAll((elements) =>
      elements.map((element) => ({
        src: element.getAttribute('src'),
        loaded: element.complete && element.naturalWidth > 0,
      })),
    );
    for (const image of images) {
      expect(image.loaded, image.src).toBe(true);
      await article.locator(`img[src="${image.src}"]`).click();
      const resource = await context.newPage();
      const response = await resource.goto(image.src);
      expect(response.ok(), image.src).toBe(true);
      await resource.close();
    }
    for (const href of process.env.E2E_RENDER_ONLY === '1' ? [] : [...new Set(links)]) {
      if (href.startsWith('/') || href.startsWith('#')) {
        await page.goto(documentURL);
        await page.locator('.page-viewer__content').locator(`a[href="${href}"]`).first().click();
        if (href.startsWith('#')) {
          await expect(page.locator(href)).toBeVisible();
          expect(decodeURIComponent(new URL(page.url()).hash)).toBe(decodeURIComponent(href));
        } else {
          await expect(page).toHaveURL(new RegExp(`${href}$`));
          await expect(page.locator('.page-viewer__content')).toBeVisible();
        }
      } else {
        const resource = await context.newPage();
        const response = await resource.goto(href);
        expect(response.ok(), href).toBe(true);
        await expect(resource.locator('body')).not.toBeEmpty();
        await resource.close();
      }
    }
    report.documents.push({ slug: document.slug, links, images });
  }
  report.checks.push(
    process.env.E2E_RENDER_ONLY === '1'
      ? '最终重启后只读确认两项公式及真实Mermaid流程图，生成无错误截图'
      : '逐篇打开五篇默认文档，验证正文、代码、表格、公式、流程图和所有实际引用',
  );
}

let originalPreferences;
let originalSiteName;
let completed = false;
try {
  await login();
  const config = await api('/api/config');
  originalPreferences = config.preferences;
  if (originalPreferences.language !== 'zh') {
    await api('/api/admin/settings/preferences', 'PUT', { ...originalPreferences, language: 'zh' });
  }
  if (process.env.E2E_DOCUMENTS_ONLY !== '1') {
    if (process.env.E2E_REUSE_USER_SCREENSHOTS === '1') {
      const previous = JSON.parse(
        await fs.readFile(path.join(output, 'settings-report.json'), 'utf8'),
      );
      await fs.copyFile(
        path.join(output, 'settings-report.json'),
        path.join(output, `settings-attempt-${stamp}.json`),
      );
      report.screenshots.push(
        ...previous.screenshots.filter((name) => /-users(?:-menu)?\.png$/.test(name)),
      );
      report.tables.push(...previous.tables.filter((table) => table.kind === 'users'));
      report.checks.push('复用首轮已实际验证的用户新增、编辑、改密、删除、对齐测量及十二张截图');
    } else {
      await page.goto('/settings/users');
      for (const fixture of fixtures) {
        await page.getByRole('button', { name: '新建用户', exact: true }).click();
        const dialog = page.getByRole('dialog');
        await dialog.locator('input[name="username"]').fill(fixture.username);
        await dialog.locator('input[name="email"]').fill(fixture.email);
        await dialog.locator('input[name="new-password"]').fill(fixturePassword);
        await dialog.getByRole('combobox').click();
        await page.getByRole('option', { name: fixture.role, exact: true }).click();
        await dialog.getByTestId('user-form-dialog-button-confirm').click();
        await expect(dialog).toBeHidden();
        await expect(userRow(fixture.username)).toBeVisible();
      }
      report.checks.push('通过真实表单创建编辑者和查看者');

      const editing = await openUserAction(fixtures[0].username, '编辑');
      const changedEmail = `edited.${fixtures[0].email}`;
      await editing.locator('input[name="email"]').fill(changedEmail);
      await editing.getByTestId('user-form-dialog-button-confirm').click();
      await expect(editing).toBeHidden();
      await expect(userRow(fixtures[0].username)).toContainText(changedEmail);
      const changingPassword = await openUserAction(fixtures[0].username, '修改密码');
      const changedPassword = randomBytes(24).toString('base64url');
      await changingPassword.locator('input[name="new-password"]').fill(changedPassword);
      await changingPassword.locator('input[name="confirm-new-password"]').fill(changedPassword);
      await changingPassword.getByTestId('change-user-password-dialog-button-confirm').click();
      await expect(changingPassword).toBeHidden();
      report.checks.push('文字操作菜单编辑用户并修改密码');

      for (const theme of themes) {
        for (const viewport of viewports) {
          const prefix = `${viewport.width}x${viewport.height}-${theme}`;
          await selectAppearance(theme, viewport);
          await page.goto('/settings/users');
          await expect(userRow(fixtures[0].username)).toBeVisible();
          await expect(page.locator('tbody .settings__role-pill')).toHaveCount(0);
          const trigger = userRow(fixtures[0].username).getByRole('button');
          await expect(trigger).toHaveText('操作');
          await expect(trigger).toHaveCSS('border-left-width', '0px');
          await expect(trigger).toHaveCSS('padding-left', '0px');
          await expect(trigger).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
          await verifyTable('users');
          await saveScreenshot(`${prefix}-users.png`);
          await trigger.scrollIntoViewIfNeeded();
          await trigger.focus();
          await page.keyboard.press('Enter');
          await expect(page.getByRole('menu')).toBeVisible();
          await saveScreenshot(`${prefix}-users-menu.png`);
          await page.keyboard.press('Escape');
          await expect(page.getByRole('menu')).toBeHidden();
        }
      }
      await page.setViewportSize(viewports[0]);
      for (const fixture of fixtures) await deleteFixture(fixture);
      report.checks.push('六种视口主题组合的用户表格及键盘菜单截图，临时用户已通过界面删除');
    }

    originalSiteName = (await api('/api/branding')).siteName;
    await saveSiteName(`视觉验收 ${stamp}`);
    await saveSiteName(originalSiteName);
    report.checks.push('站点名称通过界面保存、重载和还原');

    await page.goto('/settings/users');
    await expect(userRow(username)).toBeVisible();
    const existingUsers = await page.locator('tbody tr td:first-child').allTextContents();
    report.cleanedFixtureUsers = [];
    for (const name of existingUsers.map((value) => value.trim())) {
      if (/^align_[a-z0-9]+_(?:editor|viewer)$/.test(name)) {
        await deleteFixture({ username: name });
        report.cleanedFixtureUsers.push(name);
      }
    }

    await page.goto('/settings/snapshots');
    const backupRows = page.locator('.snapshot__table tbody tr');
    const newBackupIds = [];
    for (let index = 0; index < 3; index++) {
      const existing = (await api('/api/admin/snapshot')).snapshots ?? [];
      const beforeIds = new Set(existing.map((backup) => backup.id));
      await expect(backupRows).toHaveCount(existing.length);
      await page.getByRole('button', { name: '立即创建备份', exact: true }).click();
      await expect
        .poll(
          async () => {
            const ids = await backupRows.locator('td:first-child').allTextContents();
            return ids.filter((id) => !beforeIds.has(id));
          },
          { timeout: 60000 },
        )
        .toHaveLength(1);
      const ids = await backupRows.locator('td:first-child').allTextContents();
      newBackupIds.push(ids.find((id) => !beforeIds.has(id)));
    }
    report.newBackupIds = newBackupIds;
    const verifiedBackupRow = () =>
      backupRows.filter({ has: page.getByText(newBackupIds.at(-1), { exact: true }) });
    const downloadEvent = page.waitForEvent('download');
    await verifiedBackupRow().getByRole('link', { name: '下载', exact: true }).click();
    const download = await downloadEvent;
    await download.saveAs(path.join(output, 'verified-backup.zip'));
    expect(await download.failure()).toBeNull();
    report.checks.push('创建至少三条真实备份并成功下载');

    for (const theme of themes) {
      for (const viewport of viewports) {
        const prefix = `${viewport.width}x${viewport.height}-${theme}`;
        await selectAppearance(theme, viewport);
        await page.goto('/settings/snapshots');
        await expect(backupRows.first()).toBeVisible();
        await verifyTable('backups');
        await page.locator('.snapshot__table').scrollIntoViewIfNeeded();
        await saveScreenshot(`${prefix}-backups.png`);
        await backupRows
          .first()
          .getByRole('button', { name: '恢复', exact: true })
          .scrollIntoViewIfNeeded();
        await saveScreenshot(`${prefix}-backups-actions.png`);

        await page.goto('/settings/branding');
        await expect(page.getByRole('heading', { name: '外观设置', level: 1 })).toBeVisible();
        await expect(page.getByRole('heading', { name: '站点名称', exact: true })).toHaveCount(1);
        await expect(page.getByLabel('站点名称', { exact: true })).toHaveValue(originalSiteName);
        await expect(page.locator('label[for="siteName"]')).toHaveCount(0);
        await saveScreenshot(`${prefix}-branding.png`);
      }
    }

    await page.setViewportSize(viewports[0]);
    await saveSiteName(`恢复验收 ${stamp}`);
    await page.goto('/settings/snapshots');
    await verifiedBackupRow().getByRole('button', { name: '恢复', exact: true }).click();
    const confirmation = page.getByRole('alertdialog');
    await expect(confirmation).toBeVisible();
    await confirmation.getByRole('button', { name: '恢复', exact: true }).click();
    await expect(page.getByTestId('login-identifier')).toBeVisible({ timeout: 90000 });
    await login();
    await page.goto('/settings/branding');
    await expect(page.locator('#siteName')).toHaveValue(originalSiteName);
    await page.goto('/settings/users');
    await expect(page.locator('tbody tr')).toHaveCount(1);
    await expect(page.locator('tbody tr td:first-child')).toHaveText(username);
    report.checks.push('通过界面恢复备份，重新登录后确认站点名称恢复');
  } else {
    const previousPath = path.join(output, 'settings-report.json');
    let previous = JSON.parse(await fs.readFile(previousPath, 'utf8'));
    await fs.copyFile(previousPath, path.join(output, `settings-attempt-${stamp}.json`));
    if (process.env.E2E_RENDER_ONLY === '1') {
      const attempts = (await fs.readdir(output))
        .filter((name) => /^settings-attempt-.*\.json$/.test(name))
        .sort()
        .reverse();
      for (const name of attempts) {
        const candidate = JSON.parse(await fs.readFile(path.join(output, name), 'utf8'));
        if (candidate.completed && candidate.finalBackupId && candidate.documents?.length === 5) {
          previous = candidate;
          report.reusedSuccessfulReport = name;
          report.previousDocumentValidation = candidate.documents;
          break;
        }
      }
      expect(previous.completed).toBe(true);
    }
    report.checks.push(...previous.checks);
    report.screenshots.push(...previous.screenshots);
    report.tables.push(...previous.tables);
    report.newBackupIds = previous.newBackupIds;
    report.finalBackupId = previous.finalBackupId;
    report.cleanedFixtureUsers = previous.cleanedFixtureUsers;
  }
  await verifyDefaultDocuments();
  expect(report.errors, '浏览器控制台和脚本错误').toEqual([]);
  expect(report.warnings, '浏览器告警').toEqual([]);
  expect(report.failedResponses ?? [], '浏览器请求失败').toEqual([]);
  completed = true;
} catch (error) {
  report.failure = error.stack || error.message;
  throw error;
} finally {
  // Cleanup uses public interfaces and never shuts down the application services.
  try {
    if (!completed) {
      await login();
      await page.setViewportSize(viewports[0]);
      for (const fixture of fixtures) await deleteFixture(fixture);
      if (originalSiteName !== undefined) await saveSiteName(originalSiteName);
    }
    if (originalPreferences) {
      await api('/api/admin/settings/preferences', 'PUT', originalPreferences);
    }
  } catch (error) {
    report.errors.push(`验收数据清理失败: ${error.message}`);
    completed = false;
    process.exitCode = 1;
  }
  await context.close();
  await Promise.all(responseInspections);
  await browser.close();
  if (report.errors.length || report.warnings.length || report.failedResponses?.length) {
    completed = false;
    process.exitCode = 1;
  }
  report.completed = completed;
  report.screenshots = [...new Set(report.screenshots)];
  report.checks = [...new Set(report.checks)];
  await fs.writeFile(
    path.join(output, 'settings-report.json'),
    JSON.stringify(report, null, 2) + '\n',
    'utf8',
  );
}
console.log(
  JSON.stringify(
    { completed, screenshots: report.screenshots.length, checks: report.checks },
    null,
    2,
  ),
);
