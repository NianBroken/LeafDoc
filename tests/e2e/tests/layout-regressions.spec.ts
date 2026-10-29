import path from 'node:path';
import { expect, test } from '@playwright/test';
import { browserFetch, createPage, expectOk, pageByPathUrl } from '../helpers/api';
import LoginPage from '../pages/LoginPage';
import { toAppPath } from '../pages/appPath';

const widths = [1440, 1024, 390];
const rowHeight = 48;
const content =
  '# 选区验证\n\n中文正文与 **强调文字**、[项目地址](https://github.com/NianBroken/LeafDoc)。\n\n| 标题 | 内容 | 操作 |\n| --- | --- | --- |\n| 短内容 | ' +
  '超长内容'.repeat(50) +
  ' | 保留 |\n';
let fixturePath: string;
let fixtureId: string;

// Dedicated pages keep layout assertions independent of existing user documents.
test.beforeEach(async ({ page }) => {
  fixtureId = '';
  const login = new LoginPage(page);
  await login.goto();
  await login.login(
    process.env.E2E_ADMIN_USER || 'admin',
    process.env.E2E_ADMIN_PASSWORD || 'admin',
  );
  await page.getByTestId('user-menu-avatar').waitFor({ state: 'visible' });
  fixturePath = `layout-regression-${Date.now()}-${test.info().workerIndex}`;
  await createPage(page, { title: 'Layout regression', slug: fixturePath, content });
  const fixture = expectOk(
    await browserFetch<{ id: string }>(page, pageByPathUrl(fixturePath)),
    'load fixture',
  );
  fixtureId = fixture.id;
});

test.afterEach(async ({ page }) => {
  if (fixtureId) {
    const current = expectOk(
      await browserFetch<{ version: string }>(page, pageByPathUrl(fixturePath)),
      'load fixture version',
    );
    expectOk(
      await browserFetch(
        page,
        `/api/pages/${fixtureId}?recursive=false&version=${encodeURIComponent(current.version)}`,
        { method: 'DELETE' },
      ),
      'remove fixture',
    );
    fixtureId = '';
  }
});

test('management and Markdown tables keep equal columns, fixed rows and no horizontal overflow', async ({
  page,
}) => {
  for (const path of ['/' + fixturePath, '/settings/users']) {
    await page.goto(toAppPath(path));
    await page.locator('table').first().waitFor({ state: 'visible' });
    for (const width of widths) {
      await page.setViewportSize({ width, height: 900 });
      const tables = await page.locator('table').evaluateAll((elements) =>
        elements.map((table) => {
          const element = table as HTMLTableElement;
          return {
            width: element.clientWidth,
            scrollWidth: element.scrollWidth,
            columns: Array.from(
              element.rows[0].cells,
              (cell) => cell.getBoundingClientRect().width,
            ),
            rows: Array.from(element.rows, (row) => row.getBoundingClientRect().height),
          };
        }),
      );
      for (const table of tables) {
        expect(Math.max(...table.columns) - Math.min(...table.columns)).toBeLessThanOrEqual(1);
        expect(table.rows.every((height) => Math.abs(height - rowHeight) <= 1)).toBe(true);
        expect(table.scrollWidth).toBeLessThanOrEqual(table.width + 1);
      }
    }
  }
  await page.locator('tbody button').first().click();
  await expect(page.getByRole('menu')).toBeVisible();
});

test('system theme and native editor selection use matching readable colors', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('design-mode', 'system'));
  for (const colorScheme of ['light', 'dark'] as const) {
    await page.emulateMedia({ colorScheme });
    await page.goto(toAppPath('/' + fixturePath));
    await page.getByTestId('edit-page-button').click();
    const editor = page.locator('.cm-content');
    await editor.waitFor({ state: 'visible' });
    await editor.press('Control+Home');
    await editor.press('Control+a');
    const selection = await editor.evaluate((element) => ({
      text: window.getSelection()?.toString(),
      foreground: getComputedStyle(element).color,
      selectionForeground: getComputedStyle(element, '::selection').color,
      background: getComputedStyle(element, '::selection').backgroundColor,
      dark: document.documentElement.classList.contains('dark'),
    }));
    expect(selection.dark).toBe(colorScheme === 'dark');
    expect(selection.selectionForeground).toBe(selection.foreground);
    expect(selection.background).not.toBe('rgba(0, 0, 0, 0)');
    expect(selection.text).toContain('中文正文');
  }
});

test('viewer chrome removes only the requested icons and shortcut hint', async ({ page }) => {
  await page.goto(toAppPath('/' + fixturePath));
  await page.getByTestId('edit-page-button').waitFor({ state: 'visible' });
  await expect(page.locator('.breadcrumbs-nav__icon')).toHaveCount(0);
  await expect(page.locator('.page-viewer__favorite-toggle')).toHaveCount(0);
  const trigger = page.getByRole('button', { name: /^(跳转到页面|Go to page)$/ });
  await expect(trigger).not.toContainText('Ctrl+Alt+P');
  await expect(trigger).not.toHaveAttribute('title', /Ctrl/);
  await page.locator('.page-viewer__content h1').click();
  await page.keyboard.press('Control+Alt+p');
  await expect(page.getByRole('dialog')).toBeVisible();
});

// Uploads are restored through the public API, including existing custom resources.
test('branding previews normalize small, large and portrait images without cropping', async ({
  page,
}) => {
  const current = expectOk(
    await browserFetch<{ logoFile: string; faviconFile: string }>(page, '/api/branding'),
    'read branding',
  );
  const backups = await Promise.all(
    [current.logoFile, current.faviconFile].map(async (file) => {
      if (!file) return null;
      const response = await page.request.get('/branding/' + file);
      expect(response.ok()).toBe(true);
      return {
        name: file,
        mimeType: response.headers()['content-type'],
        buffer: await response.body(),
      };
    }),
  );
  const cookies = await page.context().cookies();
  const csrf = cookies.find((cookie) => cookie.name.endsWith('leafdoc_csrf'));
  expect(csrf).toBeDefined();
  const headers = { 'X-CSRF-Token': decodeURIComponent(csrf!.value) };
  try {
    await page.goto(toAppPath('/settings/branding'));
    await expect(page.locator('.settings__branding-asset')).toHaveCount(2);
    for (const viewportWidth of widths) {
      await page.setViewportSize({ width: viewportWidth, height: 900 });
      const centers = await page.locator('.settings__branding-asset').evaluateAll((assets) =>
        assets.map((asset) => {
          const box = asset.getBoundingClientRect();
          return box.x + box.width / 2;
        }),
      );
      expect(Math.abs(centers[0] - centers[1])).toBeLessThanOrEqual(1);
    }
    for (const [name, width] of [
      ['small', 16],
      ['large', 1600],
      ['portrait', 80],
    ] as const) {
      for (const index of [0, 1]) {
        await page
          .locator('input[type="file"]')
          .nth(index)
          .setInputFiles(path.join(__dirname, '../assets', `preview-${name}.png`));
        const image = page.locator('.settings__branding-asset img').nth(index);
        await expect
          .poll(() => image.evaluate((node) => (node as HTMLImageElement).naturalWidth))
          .toBe(width);
        await expect(image).toHaveCSS('object-fit', 'contain');
        const box = await image.boundingBox();
        expect(box?.width).toBe(index === 0 ? 128 : 64);
        expect(box?.height).toBe(64);
      }
    }
  } finally {
    for (const [index, kind] of ['logo', 'favicon'].entries()) {
      const backup = backups[index];
      const response = backup
        ? await page.request.post('/api/branding/' + kind, { headers, multipart: { file: backup } })
        : await page.request.delete('/api/branding/' + kind, { headers });
      expect(response.ok()).toBe(true);
    }
  }
});
