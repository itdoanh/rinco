import { NextRequest, NextResponse } from 'next/server';

const TRACKING_SERVICE_URL = process.env.TRACKING_SERVICE_URL || 'http://localhost:8080';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Validate basic structure
    if (!body.event || !body.tenant_id) {
      return NextResponse.json(
        { error: 'Missing required fields: event, tenant_id' },
        { status: 400 }
      );
    }

    // Enrich with server-side data
    const enrichedData = {
      ...body,
      timestamp: body.timestamp || new Date().toISOString(),
      ip_address: request.headers.get('x-forwarded-for')?.split(',')[0] || '',
      user_agent: request.headers.get('user-agent') || '',
    };

    // Forward to tracking service
    const response = await fetch(`${TRACKING_SERVICE_URL}/api/track`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(enrichedData),
    });

    if (!response.ok) {
      // Don't fail the request, just log the error
      console.warn('Tracking service error:', await response.text());
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Track error:', error);
    // Return success anyway to not block user actions
    return NextResponse.json({ success: true });
  }
}

export async function GET() {
  return NextResponse.json(
    { error: 'Method not allowed' },
    { status: 405 }
  );
}
