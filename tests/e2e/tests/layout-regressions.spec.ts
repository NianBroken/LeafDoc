import path from 'node:path';
import { expect, test } from '@playwright/test';
import { browserFetch, createPage, expectOk, pageByPathUrl } from '../helpers/api';
import LoginPage from '../pages/LoginPage';
import EditPage from '../pages/EditPage';
import MovePageDialog from '../pages/MovePageDialog';
import TreeView from '../pages/TreeView';
import ViewPage from '../pages/ViewPage';
import { toAppPath } from '../pages/appPath';

const widths = [1440, 1024, 390];
const rowHeight = 48;
const textWidths = [390, 768, 1024, 1440, 1698];
const longTitle = '长标题LongWord'.repeat(32);
const longSlug = 'long-path-'.repeat(40);
const textSamples = [
  '短标题',
  '中文内容'.repeat(100),
  'LongWord'.repeat(200),
  'Mixed 中文 text '.repeat(80),
];
const longSearchQuery = 'qwrwqew' + 'w'.repeat(350);
const longSearchTag = 'searchtag'.repeat(20);
const content =
  '# 选区验证\n\n中文正文与 **强调文字**、[项目地址](https://github.com/NianBroken/LeafDoc)。\n\n| 标题 | 内容 | 操作 |\n| --- | --- | --- |\n| 短内容 | ' +
  '超长内容'.repeat(50) +
  ' | 保留 |\n';
let fixturePath: string;
let fixtureId: string;
let extraFixturePaths: string[];

// Dedicated pages keep layout assertions independent of existing user documents.
test.beforeEach(async ({ page }) => {
  fixtureId = '';
  extraFixturePaths = [];
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
  for (const path of extraFixturePaths.reverse()) {
    const current = expectOk(
      await browserFetch<{ id: string; version: string }>(page, pageByPathUrl(path)),
      'load workflow fixture',
    );
    expectOk(
      await browserFetch(
        page,
        `/api/pages/${current.id}?recursive=false&version=${encodeURIComponent(current.version)}`,
        { method: 'DELETE' },
      ),
      'remove workflow fixture',
    );
  }
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

// Real bounding boxes catch intrinsic grid and flex overflow that DOM-only tests miss.
async function expectDialogContained(page: import('@playwright/test').Page) {
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();
  await expect
    .poll(async () =>
      dialog.evaluate((element) => {
        const box = element.getBoundingClientRect();
        return {
          horizontalBounds: box.left >= 15 && box.right <= innerWidth - 15,
          verticalBounds: box.top >= 15 && box.bottom <= innerHeight - 15,
          noHorizontalScroll: element.scrollWidth <= element.clientWidth + 1,
          fieldsContained: ![...element.querySelectorAll('input, .form-input, .dialog__path')].some(
            (child) => {
              const childBox = child.getBoundingClientRect();
              return childBox.left < box.left || childBox.right > box.right;
            },
          ),
        };
      }),
    )
    .toEqual({
      horizontalBounds: true,
      verticalBounds: true,
      noHorizontalScroll: true,
      fieldsContained: true,
    });
}

test('long text stays inside page dialogs at every viewport and low height', async ({ page }) => {
  await page.goto(toAppPath('/e/' + fixturePath));
  await page.getByTestId('edit-page-metadata-button').click();
  const title = page.getByTestId('edit-page-metadata-dialog-title-input');
  const slug = page.getByTestId('edit-page-metadata-dialog-slug-input');
  // A manually controlled slug isolates layout from asynchronous suggestions.
  await slug.fill(longSlug);
  for (const width of textWidths) {
    await page.setViewportSize({ width, height: width === 768 ? 400 : 900 });
    for (const sample of textSamples) {
      await title.fill(sample);
      await expect(title).toHaveValue(sample);
      await expectDialogContained(page);
    }
    const cancel = page.getByTestId('edit-page-metadata-dialog-button-cancel');
    await cancel.scrollIntoViewIfNeeded();
    await expect(cancel).toBeInViewport();
    await page.screenshot({ path: test.info().outputPath(`long-dialog-${width}.png`) });
  }
  await page.getByTestId('edit-page-metadata-dialog-button-cancel').click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.getByTestId('close-editor-button').click();
  const tree = new TreeView(page);
  await tree.clickRootAddButton();
  await page.getByTestId('add-page-slug-input').fill(longSlug);
  await page.getByTestId('add-page-title-input').fill(longTitle);
  await page.setViewportSize({ width: 390, height: 600 });
  await expectDialogContained(page);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.setViewportSize({ width: 1440, height: 900 });
  await new ViewPage(page).clickCopyPageButton();
  await page.getByTestId('copy-page-dialog-slug-input').fill(longSlug);
  await page.getByTestId('copy-page-dialog-title-input').fill(longTitle);
  await page.setViewportSize({ width: 390, height: 600 });
  await expectDialogContained(page);
  await page.getByTestId('copy-page-dialog-button-cancel').click();
  await page.setViewportSize({ width: 1440, height: 900 });
  const sidebarToggle = page.getByTestId('sidebar-toggle-button');
  if ((await sidebarToggle.getAttribute('aria-expanded')) !== 'true') {
    await sidebarToggle.click();
  }
  await expect(page.getByTestId('sidebar')).toBeInViewport();
  await page
    .getByTestId('sidebar')
    .getByTestId('tree-view-action-button-add-section')
    .first()
    .click();
  await page.getByTestId('add-page-slug-input').fill(longSlug);
  await page.getByTestId('add-page-title-input').fill(longTitle);
  await page.setViewportSize({ width: 390, height: 600 });
  await expectDialogContained(page);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('long text titles truncate while prose and metadata remain complete', async ({ page }) => {
  const current = expectOk(
    await browserFetch<{ version: string }>(page, pageByPathUrl(fixturePath)),
    'read long text fixture',
  );
  expectOk(
    await browserFetch(page, `/api/pages/${fixtureId}`, {
      method: 'PUT',
      body: {
        version: current.version,
        title: longTitle,
        slug: fixturePath,
        content: `# ${longTitle}\n\n${'UnbrokenText'.repeat(200)}\n\n${content}`,
        tags: ['longtag'.repeat(20)],
        properties: { long: 'PropertyValue'.repeat(100) },
      },
    }),
    'prepare long text fixture',
  );
  await page.addInitScript(() => localStorage.setItem('design-mode', 'system'));
  for (const colorScheme of ['light', 'dark'] as const) {
    await page.emulateMedia({ colorScheme });
    await page.goto(toAppPath('/' + fixturePath));
    const crumb = page.locator('.breadcrumbs-nav__current');
    await expect(crumb).toHaveText(longTitle);
    for (const width of textWidths) {
      await page.setViewportSize({ width, height: 900 });
      await expect(crumb).toHaveCSS('text-overflow', 'ellipsis');
      const layout = await page.evaluate(() => {
        const crumb = document.querySelector('.breadcrumbs-nav__current')!;
        const nav = document.querySelector('.breadcrumbs-nav')!;
        const article = document.querySelector('.page-viewer__content')!;
        return {
          pageWidth: document.documentElement.clientWidth,
          pageScroll: document.documentElement.scrollWidth,
          crumbRight: crumb.getBoundingClientRect().right,
          navRight: nav.getBoundingClientRect().right,
          truncated: crumb.scrollWidth > crumb.clientWidth,
          articleWidth: article.clientWidth,
          articleScroll: article.scrollWidth,
        };
      });
      expect(layout.pageScroll).toBeLessThanOrEqual(layout.pageWidth + 1);
      expect(layout.crumbRight).toBeLessThanOrEqual(layout.navRight + 1);
      expect(layout.truncated).toBe(true);
      expect(layout.articleScroll).toBeLessThanOrEqual(layout.articleWidth + 1);
      await expect(page.locator('.page-viewer__content h1').first()).toHaveText(longTitle);
      await page.screenshot({
        path: test.info().outputPath(`long-viewer-${colorScheme}-${width}.png`),
      });
    }
  }
});

test('long text workflows create, save, copy, move, search and reopen complete data', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto(toAppPath('/' + fixturePath));
  const tree = new TreeView(page);
  const createdSlug = fixturePath + '-created';
  const copiedSlug = fixturePath + '-copy';
  const title = 'Workflow ' + longTitle;
  const body = `# ${longTitle}\n\n${'PreservedText中文'.repeat(100)}`;
  await tree.clickRootAddButton();
  await page.getByTestId('add-page-slug-input').fill(createdSlug);
  await page.getByTestId('add-page-title-input').fill(title);
  await expectDialogContained(page);
  await page.getByTestId('add-page-dialog-button-confirm').click();
  await expect(page.locator('.cm-content')).toBeVisible();
  extraFixturePaths.push(createdSlug);
  await page.locator('.cm-content').fill(body);
  await expect
    .poll(
      async () =>
        expectOk(
          await browserFetch<{ content: string }>(page, pageByPathUrl(createdSlug)),
          'read saved content',
        ).content,
    )
    .toBe(body);
  await new EditPage(page).closeEditor();
  await page.reload();
  await expect(page.locator('.breadcrumbs-nav__current')).toHaveText(title);
  await new ViewPage(page).clickCopyPageButton();
  await page.getByTestId('copy-page-dialog-slug-input').fill(copiedSlug);
  await page.getByTestId('copy-page-dialog-title-input').fill('Copied ' + longTitle);
  await expectDialogContained(page);
  await page.getByTestId('copy-page-dialog-button-confirm').click();
  await expect(page.locator('.cm-content')).toBeVisible();
  extraFixturePaths.push(copiedSlug);
  await expect(page.locator('.cm-content')).toContainText('PreservedText中文');
  expect(
    expectOk(
      await browserFetch<{ content: string }>(page, pageByPathUrl(copiedSlug)),
      'read copied content',
    ).content,
  ).toBe(body);
  await new EditPage(page).closeEditor();
  await tree.openMoveDialogForPage('Layout regression', 'Copied ' + longTitle);
  await page.getByRole('combobox').fill('Layout regression');
  await page.getByRole('option', { name: 'Layout regression', exact: true }).click();
  await expectDialogContained(page);
  await new MovePageDialog(page).clickMoveButton();
  const movedPath = fixturePath + '/' + copiedSlug;
  await expect(page).toHaveURL(new RegExp('/' + movedPath + '$'));
  extraFixturePaths[1] = movedPath;
  await page.reload();
  await expect(page.locator('.breadcrumbs-nav__current')).toHaveText('Copied ' + longTitle);
  const moved = expectOk(
    await browserFetch<{ content: string; title: string }>(page, pageByPathUrl(movedPath)),
    'read moved page',
  );
  expect(moved.content).toBe(body);
  expect(moved.title).toBe('Copied ' + longTitle);
  await page.keyboard.press('Control+Shift+f');
  await page.getByTestId('search-input').fill('PreservedText');
  const result = page.locator('.search-result-card').filter({ hasText: 'Copied ' + longTitle });
  await expect(result).toBeVisible();
  await expect(result.locator('.search-result-card__title')).toHaveCSS('text-overflow', 'ellipsis');
  await result.click();
  await expect(page.locator('.breadcrumbs-nav__current')).toHaveText('Copied ' + longTitle);
  await page.screenshot({ path: test.info().outputPath('long-workflow-complete.png') });
});

test('long search queries and tag summaries stay inside the sidebar in all states', async ({
  page,
}) => {
  const current = expectOk(
    await browserFetch<{ version: string }>(page, pageByPathUrl(fixturePath)),
    'read search fixture',
  );
  expectOk(
    await browserFetch(page, `/api/pages/${fixtureId}`, {
      method: 'PUT',
      body: {
        version: current.version,
        title: longTitle,
        slug: fixturePath,
        content,
        tags: [longSearchTag],
        properties: {},
      },
    }),
    'prepare search fixture',
  );
  await page.goto(toAppPath('/' + fixturePath));
  const input = page.getByTestId('search-input');
  const openSearch = async () => {
    const toggle = page.getByTestId('sidebar-toggle-button');
    await expect(toggle).toBeVisible();
    if ((await toggle.getAttribute('aria-expanded')) !== 'true') await toggle.click();
    await expect(page.getByTestId('sidebar')).toBeInViewport();
    await page
      .getByTestId('sidebar')
      .getByRole('button', { name: /^(Search|搜索)$/ })
      .click();
    await expect(input).toBeVisible();
  };
  await openSearch();
  await page.getByTestId('search-tags-accordion-trigger').click();
  const tag = page.getByTestId(`tags-filter-${longSearchTag}`);
  await expect(tag).toBeVisible();
  await tag.click();
  await expect(page.getByTestId(`search-result-card-title-${fixtureId}`)).toBeVisible();
  // Query the unfiltered index as well, since an empty tag candidate set skips SQL.
  for (const query of [longSearchQuery, '中文'.repeat(180), 'mixed 中文 text '.repeat(50)]) {
    expectOk(
      await browserFetch(page, `/api/search?q=${encodeURIComponent(query)}`),
      'execute complete long search query',
    );
  }
  for (const width of textWidths) {
    await page.setViewportSize({ width, height: 900 });
    await openSearch();
    for (const query of [longSearchQuery, '中文'.repeat(180), 'mixed 中文 text '.repeat(50)]) {
      await input.fill(query);
      const summary = page.locator('.browse-results__summary-text');
      await expect(summary).toContainText(query.trim());
      await expect(summary).toHaveCSS('text-overflow', 'ellipsis');
      await expect(page.getByTestId('search-results-list')).toHaveCount(0);
      await expect
        .poll(async () =>
          page
            .locator(
              '.search, .search__body, .browse-results__toolbar, .browse-results__summary, .browse-results__summary-text, .list-view__status, .list-view__content, .browse-tags__tag-filter',
            )
            .evaluateAll((elements) =>
              elements
                .filter(
                  (element) =>
                    element.getBoundingClientRect().width > 0 &&
                    element.scrollWidth > element.clientWidth + 1 &&
                    getComputedStyle(element).textOverflow !== 'ellipsis',
                )
                .map((element) => element.className),
            ),
        )
        .toEqual([]);
      await expect(page.getByTestId('search-clear-button')).toBeInViewport();
      const inputBounds = await input.boundingBox();
      const clearBounds = await page.getByTestId('search-clear-button').boundingBox();
      expect(inputBounds).not.toBeNull();
      expect(clearBounds).not.toBeNull();
      expect(inputBounds!.x + inputBounds!.width).toBeLessThanOrEqual(clearBounds!.x);
      await expect(page.getByTestId('search-tags-clear-button')).toBeInViewport();
    }
    await page.screenshot({ path: test.info().outputPath(`long-search-${width}.png`) });
  }
  await page.getByTestId('search-clear-button').click();
  await expect(input).toHaveValue('');
  await expect(page.getByTestId(`search-result-card-title-${fixtureId}`)).toBeVisible();
  await page.getByTestId('search-tags-clear-button').click();
  await expect(page.locator('.browse-results__summary-text')).toHaveCount(0);
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
