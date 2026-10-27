import zhSettings from '../../../ui/src/shared/locales/zh/settings.json';
import frSettings from '../../../ui/src/shared/locales/fr/settings.json';
import esSettings from '../../../ui/src/shared/locales/es/settings.json';
import deSettings from '../../../ui/src/shared/locales/de/settings.json';
import enSettings from '../../../ui/src/shared/locales/en/settings.json';
import test, { expect } from '@playwright/test';
import { browserFetch } from '../helpers/api';
import PreferencesSettingsPage from '../pages/PreferencesSettingsPage';
import LoginPage from '../pages/LoginPage';
import ViewPage from '../pages/ViewPage';

const user = process.env.E2E_ADMIN_USER || 'admin';
const password = process.env.E2E_ADMIN_PASSWORD || 'admin';

// Shared instance preferences are reset through the same admin endpoint used by the UI.
test.afterEach(async ({ page }) => {
  const response = await browserFetch(page, '/api/admin/settings/preferences', {
    method: 'PUT',
    body: { language: 'en' },
  });
  expect(response.ok).toBe(true);
});

test.describe('Global preferences: language', () => {
  test('switching the language updates the UI immediately and persists across reloads', async ({
    page,
  }) => {
    const loginPage = new LoginPage(page);
    const viewPage = new ViewPage(page);
    const settingsPage = new PreferencesSettingsPage(page);

    await loginPage.goto();
    await loginPage.login(user, password);
    await viewPage.expectUserLoggedIn();

    await settingsPage.goto();
    await expect(settingsPage.pageTitle()).toHaveText(enSettings.nav.preferences);
    await expect(settingsPage.languageSelect()).toHaveText('English');

    await settingsPage.selectLanguage('Deutsch');

    // Applied immediately, without waiting for the PUT to resolve or a reload.
    await expect(settingsPage.pageTitle()).toHaveText(deSettings.nav.preferences);

    // Reload to prove this is backend-persisted, not just local UI state.
    await page.reload();
    await settingsPage.pageTitle().waitFor({ state: 'visible' });
    await expect(settingsPage.pageTitle()).toHaveText(deSettings.nav.preferences);
    await expect(settingsPage.languageSelect()).toHaveText('Deutsch');

    await settingsPage.selectLanguage('English');
    await expect(settingsPage.pageTitle()).toHaveText(enSettings.nav.preferences);
  });

  for (const { name, accountTitle } of [
    { name: 'Español', accountTitle: esSettings.nav.preferences },
    { name: 'Français', accountTitle: frSettings.nav.preferences },
    { name: '中文', accountTitle: zhSettings.nav.preferences },
  ]) {
    test(`switching to ${name} translates the preferences page`, async ({ page }) => {
      const loginPage = new LoginPage(page);
      const viewPage = new ViewPage(page);
      const settingsPage = new PreferencesSettingsPage(page);

      await loginPage.goto();
      await loginPage.login(user, password);
      await viewPage.expectUserLoggedIn();

      await settingsPage.goto();
      await expect(settingsPage.pageTitle()).toHaveText(enSettings.nav.preferences);

      await settingsPage.selectLanguage(name);
      await expect(settingsPage.pageTitle()).toHaveText(accountTitle);
      await expect(settingsPage.languageSelect()).toHaveText(name);
    });
  }
});
