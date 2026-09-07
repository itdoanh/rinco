import { NextRequest, NextResponse } from 'next/server';
import { useLandingStore } from '@/lib/store';

const LANDING_SERVICE_URL = process.env.LANDING_SERVICE_URL || 'http://localhost:8080';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ tenant: string }> }
) {
  try {
    const { tenant } = await params;

    // Try to fetch from landing service
    const response = await fetch(
      `${LANDING_SERVICE_URL}/api/pages/${tenant}/default`,
      {
        headers: {
          'X-Forwarded-For': request.headers.get('x-forwarded-for') || '',
        },
        next: { revalidate: 60 },
      }
    );

    if (!response.ok) {
      if (response.status === 404) {
        return NextResponse.json(
          { error: 'Page not found' },
          { status: 404 }
        );
      }
      throw new Error('Failed to fetch page');
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error('Page fetch error:', error);

    // Return demo data for development
    const { tenant } = await params;
    return NextResponse.json({
      id: `page-default-${tenant}`,
      slug: 'default',
      tenant_id: tenant,
      title: 'Trang chủ',
      meta_description: 'Trang chủ mặc định',
      blocks: [
        {
          id: 'hero-home',
          type: 'hero',
          data: {
            headline: `Chào mừng đến với ${tenant}`,
            subheadline: 'Nền tảng quản lý hiện đại',
            cta_text: 'Khám phá ngay',
            cta_link: '#features',
            alignment: 'center',
            theme: 'light',
          },
          order: 1,
        },
        {
          id: 'cta-home',
          type: 'cta',
          data: {
            title: 'Bắt đầu ngay hôm nay',
            description: 'Đăng ký và trải nghiệm miễn phí',
            button_text: 'Đăng ký',
            button_link: '#register',
            theme: 'gradient',
          },
          order: 2,
        },
      ],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  }
}
