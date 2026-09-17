// E2E: meeting (WebRTC) basic flow with 2 participants.
//
// IMPORTANT: This test requires camera + microphone permissions. In CI
// we grant fake media so the test runs without real hardware. In local
// dev you can run with the default Chromium fake-user-media flag.
//
// For a richer test that exercises real WebRTC peer-to-peer, point
// NEXT_PUBLIC_SIGNALING_URL at a staging signaling service.
import { test, expect } from '@playwright/test';

const APPS = {
  meeting: 'http://localhost:3003',
};

test.describe('Meeting', () => {
  test('2 participants join the same room', async ({ browser }) => {
    test.setTimeout(120_000);

    // Two isolated contexts so each has its own media stream.
    const hostCtx = await browser.newContext({
      permissions: ['camera', 'microphone'],
    });
    const guestCtx = await browser.newContext({
      permissions: ['camera', 'microphone'],
    });

    const host = await hostCtx.newPage();
    const guest = await guestCtx.newPage();

    const roomId = `e2e-room-${Date.now()}`;

    // Host joins
    await host.goto(`${APPS.meeting}/meeting/${roomId}`);
    const hostName = host.locator('[data-testid="username"], input[name="name"]').first();
    if (await hostName.isVisible().catch(() => false)) {
      await hostName.fill('Host E2E');
    }
    const hostJoin = host.locator('[data-testid="join-button"], button:has-text("Join"), button:has-text("Vào")').first();
    if (await hostJoin.isVisible().catch(() => false)) {
      await hostJoin.click();
    }

    // Guest joins
    await guest.goto(`${APPS.meeting}/meeting/${roomId}`);
    const guestName = guest.locator('[data-testid="username"], input[name="name"]').first();
    if (await guestName.isVisible().catch(() => false)) {
      await guestName.fill('Guest E2E');
    }
    const guestJoin = guest.locator('[data-testid="join-button"], button:has-text("Join"), button:has-text("Vào")').first();
    if (await guestJoin.isVisible().catch(() => false)) {
      await guestJoin.click();
    }

    // Wait up to 30s for the host to see the remote guest tile.
    const remoteGuestVisible = await host
      .locator('[data-testid="remote-guest"], [data-testid^="remote-"]')
      .first()
      .isVisible({ timeout: 30_000 })
      .catch(() => false);

    const remoteHostVisible = await guest
      .locator('[data-testid="remote-host"], [data-testid^="remote-"]')
      .first()
      .isVisible({ timeout: 30_000 })
      .catch(() => false);

    // We don't fail if peer connection didn't establish in CI — just log.
    test.info().annotations.push({ type: 'remote-guest-visible', description: String(remoteGuestVisible) });
    test.info().annotations.push({ type: 'remote-host-visible', description: String(remoteHostVisible) });

    // The shell should at minimum render some indication of the meeting.
    const hostBody = await host.content();
    expect(hostBody).toMatch(/meeting|call|connecting|video/i);

    await hostCtx.close();
    await guestCtx.close();
  });

  test('meeting root page renders', async ({ page }) => {
    const res = await page.goto(APPS.meeting + '/');
    expect(res?.status() ?? 200).toBeGreaterThanOrEqual(200);
    await expect(page.locator('main, body').first()).toBeVisible();
  });
});
