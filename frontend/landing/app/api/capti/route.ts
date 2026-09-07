import { NextRequest, NextResponse } from "next/server";

const CAPI_ACCESS_TOKEN = process.env.CAPI_ACCESS_TOKEN;
const CAPI_PIXEL_ID = process.env.CAPI_PIXEL_ID;

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Validate required credentials
    if (!CAPI_ACCESS_TOKEN || !CAPI_PIXEL_ID) {
      return NextResponse.json(
        { error: "CAPI not configured" },
        { status: 500 }
      );
    }

    // Forward to Facebook CAPI
    const response = await fetch(
      `https://graph.facebook.com/v18.0/${CAPI_PIXEL_ID}/events`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          access_token: CAPI_ACCESS_TOKEN,
          data: [
            {
              event_name: body.event_name,
              event_time: body.event_time || Math.floor(Date.now() / 1000),
              user_data: {
                em: body.user_data?.email
                  ? hashData(body.user_data.email)
                  : undefined,
                ph: body.user_data?.phone
                  ? hashData(body.user_data.phone)
                  : undefined,
                fn: body.user_data?.name
                  ? hashData(body.user_data.name)
                  : undefined,
              },
              custom_data: body.custom_data,
              data_processing_options: ["LDU"],
              data_processing_options_state: 1,
            },
          ],
        }),
      }
    );

    const data = await response.json();

    if (!response.ok) {
      console.error("CAPI error:", data);
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error("CAPI proxy error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}

// Simple hash function for CAPI (should use proper hashing in production)
function hashData(data: string): string {
  // This is a placeholder - in production use proper SHA-256 hashing
  // Facebook requires: email, phone to be SHA-256 hashed
  return data
    .toLowerCase()
    .split("")
    .map((c) => c.charCodeAt(0).toString(16))
    .join("");
}
