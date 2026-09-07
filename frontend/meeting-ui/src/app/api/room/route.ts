import { NextRequest, NextResponse } from 'next/server';

const SIGNALING_URL = process.env.SIGNALING_URL || 'http://localhost:8080';

export async function GET(request: NextRequest) {
  try {
    const roomId = request.nextUrl.searchParams.get('room_id');

    if (!roomId) {
      return NextResponse.json(
        { error: 'room_id is required' },
        { status: 400 }
      );
    }

    // Fetch room info
    const response = await fetch(`${SIGNALING_URL}/api/rooms/${roomId}`);

    if (!response.ok) {
      if (response.status === 404) {
        return NextResponse.json(
          { error: 'Room not found' },
          { status: 404 }
        );
      }
      return NextResponse.json(
        { error: 'Failed to get room info' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Room info error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { room_id, name, max_participants = 10 } = body;

    if (!room_id || !name) {
      return NextResponse.json(
        { error: 'room_id and name are required' },
        { status: 400 }
      );
    }

    // Create room
    const response = await fetch(`${SIGNALING_URL}/api/rooms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ room_id, name, max_participants }),
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to create room' },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data, { status: 201 });
  } catch (error) {
    console.error('Create room error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}

export async function DELETE(request: NextRequest) {
  try {
    const roomId = request.nextUrl.searchParams.get('room_id');

    if (!roomId) {
      return NextResponse.json(
        { error: 'room_id is required' },
        { status: 400 }
      );
    }

    // Delete room
    const response = await fetch(`${SIGNALING_URL}/api/rooms/${roomId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to delete room' },
        { status: response.status }
      );
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Delete room error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
