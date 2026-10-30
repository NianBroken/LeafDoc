import { expect } from '@playwright/test';
import { csrfHeader, loggedIn } from './apiContext';
import { toAppPath } from '../pages/appPath';

// Browser scenarios use English labels and share the same instance preferences.
const preferences = {
  language: 'en',
  dateFormat: 'iso',
  timeFormat: '24h',
};

export default async function setup() {
  const api = await loggedIn();
  let originalPreferences: unknown;
  try {
    const original = await api.get(toAppPath('/api/config'));
    expect(original.ok(), 'read original browser test preferences').toBe(true);
    originalPreferences = (await original.json()).preferences;
    const response = await api.put(toAppPath('/api/admin/settings/preferences'), {
      headers: await csrfHeader(api),
      data: preferences,
    });
    expect(response.status(), 'prepare shared browser test preferences').toBe(200);
    const config = await api.get(toAppPath('/api/config'));
    expect((await config.json()).preferences).toEqual(preferences);
  } finally {
    await api.dispose();
  }

  // Restore instance preferences even when a browser scenario fails.
  return async () => {
    const api = await loggedIn();
    try {
      const restored = await api.put(toAppPath('/api/admin/settings/preferences'), {
        headers: await csrfHeader(api),
        data: originalPreferences,
      });
      expect(restored.ok(), 'restore original browser test preferences').toBe(true);
    } finally {
      await api.dispose();
    }
  };
}
