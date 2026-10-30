// Automatic saving is always active and has no configurable preference.
import test, { expect } from '@playwright/test';
import { browserFetch, createPage, expectOk, pageByPathUrl } from '../helpers/api';
import PreferencesSettingsPage from '../pages/PreferencesSettingsPage';
import LoginPage from '../pages/LoginPage';
import ViewPage from '../pages/ViewPage';

const user = process.env.E2E_ADMIN_USER || 'admin';
const password = process.env.E2E_ADMIN_PASSWORD || 'admin';

test('automatically saves edits without exposing a preference switch', async ({ page }) => {
  const login = new LoginPage(page);
  const viewer = new ViewPage(page);
  const settings = new PreferencesSettingsPage(page);
  const slug = `always-save-${Date.now()}`;
  let id: string | undefined;
  await login.goto();
  await login.login(user, password);
  await viewer.expectUserLoggedIn();
  try {
    await settings.goto();
    await expect(page.locator('[data-testid="preferences-panel"] [role="checkbox"]')).toHaveCount(
      0,
    );
    await createPage(page, { title: 'Automatic save verification', slug, content: 'initial' });
    const original = expectOk(
      await browserFetch<{ id: string }>(page, pageByPathUrl(slug)),
      'read created page',
    );
    id = original.id;
    await viewer.goto(`/${slug}`);
    await viewer.clickEditPageButton();
    const editor = page.locator('.cm-content');
    await expect(editor).toBeVisible();
    await editor.fill('always saved automatically');
    await expect
      .poll(async () => {
        const result = expectOk(
          await browserFetch<{ content: string }>(page, pageByPathUrl(slug)),
          'read saved page',
        );
        return result.content;
      })
      .toContain('always saved automatically');
    await page.reload();
    await expect(page.locator('.cm-content')).toContainText('always saved automatically');
  } finally {
    if (id) {
      const saved = expectOk(
        await browserFetch<{ version: string }>(page, `/api/pages/${id}`),
        'read cleanup version',
      );
      expectOk(
        await browserFetch(page, `/api/pages/${id}?version=${encodeURIComponent(saved.version)}`, {
          method: 'DELETE',
        }),
        'delete verification page',
      );
    }
  }
});
