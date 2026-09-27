import { expect, test } from '@playwright/test';

// The sandbox, end to end: break a simulated site, watch the monitor open an incident by itself,
// post an update, fix the site, and watch the incident close.
test('a visitor breaks a site and watches the incident open and close', async ({ page }) => {
  await page.goto('/console/');
  await page.getByRole('button', { name: 'Start a sandbox →' }).click();
  await expect(page).toHaveURL(/\/console\/monitors$/);
  await expect(page.getByRole('note')).toContainText('Everything here is simulated');

  const checkout = page.getByRole('row', { name: /Checkout API/ });
  await checkout.getByLabel('Down').check({ force: true });
  await expect(checkout.getByRole('link', { name: 'open incident' })).toBeVisible({ timeout: 60_000 });

  await checkout.getByRole('link', { name: 'open incident' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Checkout API is down');
  await page.getByPlaceholder('What changed?').fill('Failing over to the backup provider.');
  await page.getByRole('button', { name: 'Post' }).click();
  await expect(page.getByText('Failing over to the backup provider.')).toBeVisible();

  await page.goto('/console/monitors');
  await page.getByRole('row', { name: /Checkout API/ }).getByLabel('Up').check({ force: true });
  await page.goto('/console/incidents');
  await expect(page.locator('.pill.resolved').first()).toBeVisible({ timeout: 60_000 });
});

test('the console is complete at phone width, with no sideways scrolling', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/console/');
  await page.getByRole('button', { name: 'Start a sandbox →' }).click();
  await expect(page.getByRole('row', { name: /Checkout API/ })).toBeVisible();
  for (const path of ['/console/monitors', '/console/incidents', '/console/status']) {
    await page.goto(path);
    await page.waitForLoadState('networkidle');
    const [scroll, width] = await page.evaluate(() => [document.documentElement.scrollWidth, document.documentElement.clientWidth]);
    expect(scroll, path).toBeLessThanOrEqual(width + 1);
  }
});

test('sandboxes are isolated and never see the owner area', async ({ page, browser }) => {
  await page.goto('/console/');
  await page.getByRole('button', { name: 'Start a sandbox →' }).click();
  await expect(page.getByRole('row', { name: /Checkout API/ })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Readers' })).toHaveCount(0);
  await page.goto('/console/readers');
  await expect(page).toHaveURL(/\/console\/monitors$/);
  expect((await page.request.get('/api/analytics')).status()).toBe(403);

  // Another visitor's sandbox doesn't contain this one's changes.
  await page.getByRole('row', { name: /Search/ }).getByLabel('Down').check({ force: true });
  const other = await browser.newPage();
  await other.goto('/console/');
  await other.getByRole('button', { name: 'Start a sandbox →' }).click();
  const search = other.getByRole('row', { name: /Search/ });
  await expect(search.getByLabel('Slow')).toBeChecked();
});
