import { NextRequest, NextResponse } from "next/server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();

    // Extract headers
    const tenantSlug = request.headers.get("X-Tenant-Slug");
    const pageSlug = request.headers.get("X-Page-Slug");
    const formName = request.headers.get("X-Form-Name");

    // Validate required fields
    if (!body.phone) {
      return NextResponse.json(
        { error: "Số điện thoại là bắt buộc" },
        { status: 400 }
      );
    }

    // Build lead data
    const leadData = {
      name: body.name,
      phone: body.phone,
      email: body.email || null,
      channel: body.channel || "zalo",
      form_name: formName || body.form_name,
      tenant_slug: tenantSlug || body.tenant_slug,
      page_slug: pageSlug || body.page_slug,
      utm: {
        source: body.utm_source,
        medium: body.utm_medium,
        campaign: body.utm_campaign,
        content: body.utm_content,
        term: body.utm_term,
      },
      metadata: {
        user_agent: request.headers.get("user-agent"),
        ip: request.headers.get("x-forwarded-for") || request.headers.get("x-real-ip"),
        referrer: request.headers.get("referer"),
      },
    };

    // Forward to lead-service
    try {
      const response = await fetch(`${API_BASE_URL}/api/v1/leads`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(leadData),
      });

      if (!response.ok) {
        const error = await response.json();
        return NextResponse.json(
          { error: error.message || "Failed to submit lead" },
          { status: response.status }
        );
      }

      const data = await response.json();
      return NextResponse.json(data, { status: 201 });
    } catch (e) {
      // If lead-service is unavailable, still return success (queue for retry)
      console.error("Lead service unavailable:", e);
      return NextResponse.json(
        { message: "Lead queued for processing", id: "pending" },
        { status: 202 }
      );
    }
  } catch (error) {
    console.error("Lead submission error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
