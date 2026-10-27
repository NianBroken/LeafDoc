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
    body: { dateFormat: 'iso', timeFormat: '24h' },
  });
  expect(response.ok).toBe(true);
});

test.describe('Global preferences: date & time format', () => {
  test('picking a date/time format persists across reloads', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const viewPage = new ViewPage(page);
    const settingsPage = new PreferencesSettingsPage(page);

    await loginPage.goto();
    await loginPage.login(user, password);
    await viewPage.expectUserLoggedIn();

    await settingsPage.goto();
    await expect(settingsPage.dateFormatSelect()).toHaveText('2026-08-27');
    await expect(settingsPage.timeFormatSelect()).toHaveText('24-hour (14:30)');

    await settingsPage.selectDateFormat('27.08.2026');
    await settingsPage.selectTimeFormat('24-hour (14:30)');

    await page.reload();
    await settingsPage.dateFormatSelect().waitFor({ state: 'visible' });
    await expect(settingsPage.dateFormatSelect()).toHaveText('27.08.2026');
    await expect(settingsPage.timeFormatSelect()).toHaveText('24-hour (14:30)');
  });
});
