import { expect, type Page } from '@playwright/test';

// CSS chunks retain the ordering declared by the production HTML entry point.
export async function loadProductionStyles(page: Page): Promise<string> {
  const response = await page.request.get('/');
  expect(response.ok()).toBe(true);
  const html = await response.text();
  const links = [...html.matchAll(/<link\b[^>]*\brel=["']stylesheet["'][^>]*>/g)];
  expect(links.length, 'The served frontend includes production stylesheets').toBeGreaterThan(0);
  return links
    .map(([link]) => {
      const href = link.match(/\bhref=["']([^"']+)["']/)?.[1];
      expect(href).toBeTruthy();
      return `<link rel="stylesheet" href="${new URL(href!, response.url()).href}">`;
    })
    .join('');
}
