// Real browser acceptance uses temporary fixtures and restores instance settings.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomBytes } from 'node:crypto';
import { chromium, expect as assertion } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
const ports = JSON.parse(
  await fs.readFile(path.join(root, 'internal/core/serviceports/ports.json'), 'utf8'),
);
const baseURL = process.env.E2E_BASE_URL || `https://localhost:${ports.frontend}`;
const output = path.join(root, 'build/verification');
const expect = assertion.configure({ timeout: 15000 });
const report = { checks: [], widths: [], errors: [], warnings: [], screenshots: [] };
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL, viewport: { width: 1440, height: 900 } });
const page = await context.newPage();
const fixtureUsers = [];
const fixturePages = [];
const stamp = Date.now().toString(36);
const fixturePassword = randomBytes(24).toString('base64url');
let originalBranding;
let originalPreferences;
let originalPublic;

page.on('pageerror', (error) => report.errors.push(error.message));
page.on('console', (message) => {
  if (message.type() === 'error') report.errors.push(message.text());
  if (message.type() === 'warning') report.warnings.push(message.text());
});
async function api(ctx, route, method = 'GET', data, expected) {
  const cookies = await ctx.cookies();
  const csrf = cookies.find((cookie) => cookie.name.endsWith('leafdoc_csrf'));
  const response = await ctx.request.fetch(route, {
    method,
    data,
    headers: csrf ? { 'X-CSRF-Token': decodeURIComponent(csrf.value) } : {},
  });
  if (expected) expect(response.status(), `${method} ${route}`).toBe(expected);
  else expect(response.ok(), `${method} ${route}: ${await response.text()}`).toBe(true);
  const raw = await response.text();
  return raw ? JSON.parse(raw) : undefined;
}
async function login(ctx, identifier, password) {
  await api(ctx, '/api/config');
  return api(ctx, '/api/auth/login', 'POST', { identifier, password });
}
async function removePage(id) {
  const current = await api(context, `/api/pages/${id}`);
  await api(context, `/api/pages/${id}?version=${encodeURIComponent(current.version)}`, 'DELETE');
}
async function screenshot(name) {
  await page.screenshot({ path: path.join(output, name), fullPage: true, animations: 'disabled' });
  report.screenshots.push(name);
}
try {
  await fs.mkdir(output, { recursive: true });
  const config = await api(context, '/api/config');
  originalPreferences = config.preferences;
  originalPublic = config.publicAccess;
  originalBranding = await api(context, '/api/branding');
  await page.goto('/login');
  await page.getByTestId('login-identifier').fill(process.env.E2E_ADMIN_USER);
  await page.getByTestId('login-password').fill(process.env.E2E_ADMIN_PASSWORD);
  await page.getByTestId('login-submit').click();
  await expect(page.getByTestId('user-menu-avatar')).toBeVisible();
  report.checks.push('管理员通过真实登录表单登录');

  // Remove only fixtures owned by this acceptance test's documented prefix.
  const tree = await api(context, '/api/tree');
  const flatten = (nodes) => nodes.flatMap((node) => [node, ...flatten(node.children || [])]);
  for (const item of flatten([tree]))
    if (/^always-save-\d+$/.test(item.slug)) await removePage(item.id);

  const sections = [
    'preferences',
    'account',
    'branding',
    'public-access',
    'toc-display',
    'users',
    'snapshots',
    'importer',
    'brokenLinks',
  ];
  for (const viewport of [
    { width: 1698, height: 965 },
    { width: 1440, height: 900 },
    { width: 1024, height: 768 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    const measurements = [];
    for (const section of sections) {
      await page.goto(`/settings/${section}`);
      await expect(page.locator('h1.settings__title')).toBeVisible();
      if (section === 'users') await expect(page.locator('tbody')).toContainText('admin');
      await expect(page.locator('.settings-nav .list-view__header')).toHaveCount(0);
      await expect(page.locator('[data-testid="preferences-panel"] [role="checkbox"]')).toHaveCount(
        0,
      );
      const box = await page.locator('.settings').boundingBox();
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth);
      expect(overflow, `${section} page overflow`).toBe(false);
      measurements.push({ section, x: box.x, width: box.width });
      if (
        ['account', 'users', 'brokenLinks'].includes(section) &&
        [1440, 390].includes(viewport.width)
      )
        await screenshot(`${section}-${viewport.width}.png`);
    }
    const reference = measurements[0];
    for (const measure of measurements) {
      expect(Math.abs(measure.x - reference.x), `${measure.section} left edge`).toBeLessThanOrEqual(
        1,
      );
      expect(
        Math.abs(measure.width - reference.width),
        `${measure.section} width`,
      ).toBeLessThanOrEqual(1);
    }
    report.widths.push({ viewport, measurements });
  }
  report.checks.push('九类设置页在四种窗口尺寸下宽度一致，页面无横向溢出');

  const specialName = '中文 <站点> "名称" $&';
  await api(context, '/api/branding', 'PUT', { siteName: specialName });
  for (const route of ['/', '/settings/preferences', '/settings/users']) {
    const response = await context.request.get(route);
    const html = await response.text();
    expect(html).not.toContain('{{__');
    expect(html).toContain('leafdoc-branding');
    expect(html).toContain('中文 &lt;站点&gt;');
    await page.goto(route, { waitUntil: 'domcontentloaded' });
    await expect.poll(() => page.title()).toContain(specialName);
    expect(
      await page.evaluate(
        () => JSON.parse(document.querySelector('meta[name="leafdoc-branding"]').content).siteName,
      ),
    ).toBe(specialName);
  }
  await api(context, '/api/branding', 'PUT', { siteName: originalBranding.siteName });
  report.checks.push('中文与特殊字符站点名在初始 HTML、初始化数据及浏览器标题中一致');

  await page.setViewportSize({ width: 1440, height: 900 });
  const slug = `runtime-save-${stamp}`;
  const created = await api(context, '/api/pages', 'POST', {
    parentId: null,
    title: '自动保存验收',
    slug,
    kind: 'page',
  });
  fixturePages.push(created.id);
  await page.goto(`/e/${slug}`);
  await expect(page.locator('.cm-content')).toBeVisible();
  await page.locator('.cm-content').fill('自动保存验收内容');
  await expect
    .poll(async () => (await api(context, `/api/pages/${created.id}`)).content)
    .toContain('自动保存验收内容');
  await page.reload();
  await expect(page.locator('.cm-content')).toContainText('自动保存验收内容');
  await screenshot('automatic-save.png');
  report.checks.push('编辑器自动保存并在刷新后保留正文');

  for (const role of ['editor', 'viewer']) {
    const username = `runtime_${stamp}_${role}`;
    await api(context, '/api/users', 'POST', {
      username,
      email: `${username}@example.com`,
      password: fixturePassword,
      role,
    });
    const users = await api(context, '/api/users');
    fixtureUsers.push(users.find((user) => user.username === username).id);
    const roleContext = await browser.newContext({ baseURL });
    try {
      await login(roleContext, username, fixturePassword);
      await api(roleContext, '/api/users', 'GET', undefined, 403);
      const current = await api(roleContext, `/api/pages/${created.id}`);
      const data = {
        version: current.version,
        title: current.title,
        slug,
        content: current.content + '\n角色验收',
        tags: [],
        properties: {},
      };
      if (role === 'editor') await api(roleContext, `/api/pages/${created.id}`, 'PUT', data);
      else await api(roleContext, `/api/pages/${created.id}`, 'PUT', data, 403);
      await api(roleContext, '/api/auth/logout', 'POST');
      await api(
        roleContext,
        '/api/pages',
        'POST',
        { parentId: null, title: 'Denied', slug: 'denied', kind: 'page' },
        401,
      );
    } finally {
      await roleContext.close();
    }
  }
  report.checks.push('编辑者可写，查看者不可写，两者均不能管理用户，退出后写入被拒绝');
  expect(report.errors).toEqual([]);
  expect(report.warnings).toEqual([]);
} finally {
  try {
    await page.goto('/settings/account');
    if (originalBranding)
      await api(context, '/api/branding', 'PUT', { siteName: originalBranding.siteName });
    for (const id of fixturePages) await removePage(id);
    for (const id of fixtureUsers) await api(context, `/api/users/${id}`, 'DELETE');
    if (originalPreferences)
      await api(context, '/api/admin/settings/preferences', 'PUT', originalPreferences);
    if (originalPublic !== undefined)
      expect((await api(context, '/api/config')).publicAccess).toBe(originalPublic);
  } finally {
    await fs.writeFile(
      path.join(output, 'runtime-report.json'),
      JSON.stringify(report, null, 2),
      'utf8',
    );
    await browser.close();
  }
}
console.log(JSON.stringify(report));
