import { defineConfig } from '@playwright/test';
import { e2eBaseUrl } from './helpers/servicePorts';

const baseURL = e2eBaseUrl;
const headless = process.env.E2E_HEADLESS !== 'false';
const maxFailures = Number(process.env.E2E_MAX_FAILURES || '1');

export default defineConfig({
  globalSetup: './helpers/setup.ts',
  testDir: './tests',
  timeout: 3 * 60 * 1000, // 3 minutes per test
  maxFailures,
  use: {
    baseURL,
    headless,
    channel: process.env.E2E_BROWSER_CHANNEL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
});
