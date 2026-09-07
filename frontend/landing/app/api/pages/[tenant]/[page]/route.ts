import { NextRequest, NextResponse } from 'next/server';
import { useLandingStore } from '@/lib/store';

const LANDING_SERVICE_URL = process.env.LANDING_SERVICE_URL || 'http://localhost:8080';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ tenant: string; page: string }> }
) {
  try {
    const { tenant, page } = await params;

    // Try to fetch from landing service
    const response = await fetch(
      `${LANDING_SERVICE_URL}/api/pages/${tenant}/${page}`,
      {
        headers: {
          'X-Forwarded-For': request.headers.get('x-forwarded-for') || '',
        },
        next: { revalidate: 60 }, // Cache for 60 seconds
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
    const { tenant, page: pageSlug } = await params;
    return NextResponse.json({
      id: `page-${pageSlug}`,
      slug: pageSlug,
      tenant_id: tenant,
      title: 'Trang Demo',
      meta_description: 'Trang demo cho mục đích phát triển',
      blocks: getDemoBlocks(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  }
}

function getDemoBlocks() {
  return [
    {
      id: 'hero-1',
      type: 'hero',
      data: {
        headline: 'Chào mừng đến với RINCO',
        subheadline: 'Nền tảng quản lý doanh nghiệp hàng đầu Việt Nam',
        cta_text: 'Bắt đầu ngay',
        cta_link: '#contact',
        alignment: 'center',
        theme: 'light',
      },
      order: 1,
    },
    {
      id: 'stats-1',
      type: 'stats',
      data: {
        title: 'Số liệu ấn tượng',
        stats: [
          { value: '1000', label: 'Khách hàng', suffix: '+' },
          { value: '99.9', label: 'Uptime', suffix: '%' },
          { value: '50', label: 'Quốc gia', suffix: '+' },
          { value: '24/7', label: 'Hỗ trợ' },
        ],
        theme: 'light',
      },
      order: 2,
    },
    {
      id: 'features-1',
      type: 'feature_grid',
      data: {
        title: 'Tính năng nổi bật',
        subtitle: 'Giải pháp toàn diện cho doanh nghiệp của bạn',
        columns: 3,
        features: [
          { icon: 'zap', title: 'Tốc độ nhanh', description: 'Hiệu suất vượt trội với công nghệ hiện đại' },
          { icon: 'shield', title: 'Bảo mật', description: 'Mã hóa dữ liệu đầu cuối an toàn tuyệt đối' },
          { icon: 'users', title: 'Đội ngũ', description: 'Hỗ trợ chuyên nghiệp 24/7' },
        ],
      },
      order: 3,
    },
    {
      id: 'cta-1',
      type: 'cta',
      data: {
        title: 'Sẵn sàng bắt đầu?',
        description: 'Đăng ký ngay hôm nay và trải nghiệm miễn phí 14 ngày',
        button_text: 'Đăng ký ngay',
        button_link: '#contact',
        theme: 'gradient',
      },
      order: 4,
    },
  ];
}
