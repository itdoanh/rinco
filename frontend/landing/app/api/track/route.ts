import { NextRequest, NextResponse } from "next/server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Add server-side metadata
    const eventData = {
      ...body,
      timestamp: body.timestamp || Date.now(),
      server_timestamp: Date.now(),
      metadata: {
        user_agent: request.headers.get("user-agent"),
        ip: request.headers.get("x-forwarded-for") || request.headers.get("x-real-ip"),
        referrer: request.headers.get("referer"),
        url: body.properties?.url || request.url,
      },
    };

    // Forward to tracking service
    try {
      await fetch(`${API_BASE_URL}/api/v1/track/events`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(eventData),
      });
    } catch (e) {
      // Log but don't fail
      console.error("Tracking service unavailable:", e);
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Tracking error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
