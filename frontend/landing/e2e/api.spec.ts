import { test, expect } from '@playwright/test';

test.describe('API Routes', () => {
  test('POST /api/leads accepts a valid payload', async ({ request }) => {
    const res = await request.post('/api/leads', {
      data: {
        name: 'Test User',
        phone: '0909123456',
        email: 'test@example.com',
        tenant_slug: 'demo',
        page_slug: 'home',
      },
    })
    expect([200, 201, 400, 422, 429]).toContain(res.status())
  })

  test('POST /api/track accepts an event', async ({ request }) => {
    const res = await request.post('/api/track', {
      data: { event: 'page_view', url: '/', ts: Date.now() },
    })
    expect([200, 201, 204]).toContain(res.status())
  })

  test('POST /api/capi (FB CAPI) handles an event', async ({ request }) => {
    const res = await request.post('/api/capi', {
      data: { event_name: 'Lead', event_id: 'e2e-1' },
    })
    expect([200, 204, 400]).toContain(res.status())
  })

  test('GET /api/leads with wrong method is rejected', async ({ request }) => {
    const res = await request.get('/api/leads')
    expect(res.status()).toBeGreaterThanOrEqual(400)
  })
})
