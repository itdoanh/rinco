import { NextRequest, NextResponse } from "next/server";

// Landing service tracks events via landing-service (port 8086).
const LANDING_SERVICE_URL = process.env.LANDING_SERVICE_URL || "http://localhost:8086";

export async function POST(request: NextRequest) {
  let body: any;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      { error: "Invalid JSON body" },
      { status: 400 }
    );
  }

  if (!body || typeof body !== "object" || Array.isArray(body)) {
    return NextResponse.json(
      { error: "Body must be a JSON object" },
      { status: 400 }
    );
  }

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

  // Fire-and-forget to landing-service tracking endpoint.
  // Always return 200 so a missing backend never breaks the client.
  fetch(`${LANDING_SERVICE_URL}/api/v1/track/event`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(eventData),
  }).catch(() => {});

  return NextResponse.json({ success: true });
}
