import { NextRequest, NextResponse } from 'next/server';

const LANDING_SERVICE_URL = process.env.LANDING_SERVICE_URL || 'http://localhost:8080';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Validate required fields
    if (!body.event || !body.pixel_id) {
      return NextResponse.json(
        { error: 'Missing required fields: event, pixel_id' },
        { status: 400 }
      );
    }

    // Forward to landing service for CAPI processing
    const response = await fetch(`${LANDING_SERVICE_URL}/api/capi`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Forwarded-For': request.headers.get('x-forwarded-for') || '',
      },
      body: JSON.stringify({
        ...body,
        ip_address: request.headers.get('x-forwarded-for')?.split(',')[0] || '',
        user_agent: request.headers.get('user-agent') || '',
      }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      return NextResponse.json(
        { error: errorData.message || 'CAPI submission failed' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: 200 });
  } catch (error) {
    console.error('CAPI error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function GET() {
  return NextResponse.json(
    { error: 'Method not allowed' },
    { status: 405 }
  );
}
