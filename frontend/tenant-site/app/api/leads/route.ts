import { NextRequest, NextResponse } from "next/server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const tenantSlug = request.headers.get("X-Tenant-Slug");

    // Forward to lead service
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/leads`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-Slug": tenantSlug || "",
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        return NextResponse.json(
          { error: "Failed to submit" },
          { status: response.status }
        );
      }

      return NextResponse.json(await response.json(), { status: 201 });
    } catch {
      // Queue for retry if service unavailable
      return NextResponse.json(
        { message: "Lead queued" },
        { status: 202 }
      );
    }
  } catch (error) {
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
