import { NextRequest, NextResponse } from 'next/server';

const SIGNALING_URL = process.env.SIGNALING_URL || 'http://localhost:8080';

export async function GET(request: NextRequest) {
  try {
    const roomId = request.nextUrl.searchParams.get('room_id');
    const userId = request.nextUrl.searchParams.get('user_id');

    if (!roomId) {
      return NextResponse.json(
        { error: 'room_id is required' },
        { status: 400 }
      );
    }

    // Fetch signaling status
    const response = await fetch(`${SIGNALING_URL}/api/signaling/status?room_id=${roomId}`);

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to get signaling status' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Signaling status error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { action, room_id, user_id, data } = body;

    if (!action || !room_id) {
      return NextResponse.json(
        { error: 'action and room_id are required' },
        { status: 400 }
      );
    }

    // Forward to signaling service
    const response = await fetch(`${SIGNALING_URL}/api/signaling`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, room_id, user_id, data }),
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to process signaling' },
        { status: response.status }
      );
    }

    const responseData = await response.json();
    return NextResponse.json(responseData);
  } catch (error) {
    console.error('Signaling error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
