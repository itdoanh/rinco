'use client';

import { useEffect, useState } from 'react';
import LeadForm from '@/components/LeadForm';

declare global {
  interface Window {
    fbq?: (...args: any[]) => void;
    gtag?: (...args: any[]) => void;
  }
}

export default function HomePage() {
  const [scrollY, setScrollY] = useState(0);
  const [timeLeft, setTimeLeft] = useState({ days: 0, hours: 0, minutes: 0, seconds: 0 });

  useEffect(() => {
    // Scroll listener for parallax
    const handleScroll = () => setScrollY(window.scrollY);
    window.addEventListener('scroll', handleScroll);

    // Countdown timer
    const target = new Date();
    target.setDate(target.getDate() + 7); // 7 days from now
    const tick = () => {
      const now = new Date();
      const diff = target.getTime() - now.getTime();
      const days = Math.floor(diff / (1000 * 60 * 60 * 24));
      const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);
      setTimeLeft({ days, hours, minutes, seconds });
    };
    const interval = setInterval(tick, 1000);
    tick();

    // Initialize AOS
    if (typeof window !== 'undefined') {
      import('aos').then((AOS) => {
        AOS.init({ duration: 800, once: true });
      });
    }

    // Initialize Facebook Pixel (replace with real pixel ID in prod)
    if (typeof window !== 'undefined' && !window.fbq) {
      const fbPixelId = process.env.NEXT_PUBLIC_FB_PIXEL_ID;
      if (fbPixelId) {
        const script = document.createElement('script');
        script.innerHTML = `
          !function(f,b,e,v,n,t,s)
          {if(f.fbq)return;n=f.fbq=function(){n.callMethod?
          n.callMethod.apply(n,arguments):n.queue.push(arguments)};
          if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
          n.queue=[];t=b.createElement(e);t.async=!0;
          t.src=v;s=b.getElementsByTagName(e)[0];
          s.parentNode.insertBefore(t,s)}(window, document,'script',
          'https://connect.facebook.net/en_US/fbevents.js');
          fbq('init', '${fbPixelId}');
          fbq('track', 'PageView');
        `;
        document.head.appendChild(script);
      }
    }

    // Track page view
    if (typeof window !== 'undefined' && window.fbq) {
      window.fbq('track', 'PageView');
    }

    return () => {
      window.removeEventListener('scroll', handleScroll);
      clearInterval(interval);
    };
  }, []);

  return (
    <main className="min-h-screen bg-navy-900">
      {/* HERO SECTION */}
      <section className="relative min-h-screen flex items-center justify-center overflow-hidden px-4 py-16">
        {/* Background gradient */}
        <div
          className="absolute inset-0 opacity-30"
          style={{
            background:
              'radial-gradient(circle at 50% 50%, rgba(245,166,35,0.15) 0%, transparent 60%)',
            transform: `translateY(${scrollY * 0.3}px)`,
          }}
        />

        <div className="relative max-w-6xl mx-auto grid lg:grid-cols-2 gap-12 items-center">
          {/* Left content */}
          <div className="space-y-6" data-aos="fade-right">
            <div className="inline-flex items-center gap-2 bg-gold/10 border border-gold/30 rounded-full px-4 py-2">
              <span className="w-2 h-2 bg-gold rounded-full animate-pulse"></span>
              <span className="text-gold text-sm font-medium">Webinar Miễn Phí - 100 NĐT Đầu Tiên</span>
            </div>

            <h1 className="text-4xl md:text-5xl lg:text-6xl font-display font-extrabold leading-tight">
              Tối Ưu <span className="text-gold">Dòng Tiền 2026</span> Với Hàng Hóa Phái Sinh
            </h1>

            <p className="text-lg md:text-xl text-gray-300 leading-relaxed">
              Giải mã cơ chế <strong className="text-gold">T+0</strong> & sinh lời 2 chiều. Cùng{' '}
              <strong>Chuyên gia MXV</strong> chia sẻ bí quyết từ những trader hàng đầu Việt Nam.
            </p>

            <div className="flex flex-wrap gap-4 pt-4">
              <div className="flex items-center gap-2">
                <span className="text-gold text-2xl">✓</span>
                <span>Live Q&A với chuyên gia</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-gold text-2xl">✓</span>
                <span>Tặng 10 Ebook Thực Chiến</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-gold text-2xl">✓</span>
                <span>Hoàn toàn miễn phí</span>
              </div>
            </div>
          </div>

          {/* Right form */}
          <div data-aos="fade-left">
            <LeadForm />
          </div>
        </div>
      </section>

      {/* COUNTDOWN SECTION */}
      <section className="bg-navy-800 py-12 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <h3 className="text-2xl md:text-3xl font-bold mb-2">Webinar diễn ra trong</h3>
          <p className="text-gray-400 mb-8">Đừng bỏ lỡ cơ hội tham gia cùng chuyên gia</p>

          <div className="grid grid-cols-4 gap-3 md:gap-6 max-w-2xl mx-auto">
            {[
              { value: timeLeft.days, label: 'Ngày' },
              { value: timeLeft.hours, label: 'Giờ' },
              { value: timeLeft.minutes, label: 'Phút' },
              { value: timeLeft.seconds, label: 'Giây' },
            ].map((item, i) => (
              <div key={i} className="bg-navy-700 border border-gold/20 rounded-xl p-4 md:p-6">
                <div className="text-3xl md:text-5xl font-bold text-gold">
                  {String(item.value).padStart(2, '0')}
                </div>
                <div className="text-xs md:text-sm text-gray-400 mt-2">{item.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* FEATURES SECTION */}
      <section className="py-20 px-4">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl md:text-5xl font-display font-bold text-center mb-4">
            Bạn Sẽ Học Được Gì?
          </h2>
          <p className="text-center text-gray-400 mb-16 max-w-2xl mx-auto">
            4 nội dung chi tiết được chia sẻ bởi các chuyên gia từ MXV và APEX Fintech
          </p>

          <div className="grid md:grid-cols-2 gap-6">
            {[
              {
                icon: '📊',
                title: 'Cơ chế T+0 & Sinh lời 2 chiều',
                desc: 'Hiểu rõ cách thức giao dịch hàng hóa phái sinh với cơ chế thanh toán trong ngày.',
              },
              {
                icon: '💰',
                title: 'Quản lý vốn thông minh',
                desc: 'Chiến lược phân bổ vốn, quản lý rủi ro và tối ưu lợi nhuận bền vững.',
              },
              {
                icon: '🎯',
                title: 'Phân tích kỹ thuật chuyên sâu',
                desc: 'Ứng dụng các chỉ báo kỹ thuật vào thị trường hàng hóa phái sinh Việt Nam.',
              },
              {
                icon: '📚',
                title: 'Bộ 10 Ebook Thực Chiến',
                desc: 'Tài liệu độc quyền tổng hợp kinh nghiệm từ các trader chuyên nghiệp.',
              },
            ].map((f, i) => (
              <div
                key={i}
                data-aos="fade-up"
                data-aos-delay={i * 100}
                className="bg-navy-800 border border-gray-700 hover:border-gold rounded-2xl p-8 transition-all hover:scale-105"
              >
                <div className="text-5xl mb-4">{f.icon}</div>
                <h3 className="text-xl font-bold mb-3 text-gold">{f.title}</h3>
                <p className="text-gray-300 leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* EXPERT SECTION */}
      <section className="bg-navy-800 py-20 px-4">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl md:text-5xl font-display font-bold mb-4">
            Diễn Giả <span className="text-gold">Uy Tín</span>
          </h2>
          <p className="text-gray-400 mb-12">Chuyên gia hàng đầu từ MXV & APEX Fintech</p>

          <div className="grid md:grid-cols-2 gap-8">
            {[
              { name: 'Nguyễn Văn A', role: 'Chuyên gia MXV', exp: '15+ năm kinh nghiệm' },
              { name: 'Trần Thị B', role: 'CEO APEX Fintech', exp: '10+ năm đào tạo' },
            ].map((p, i) => (
              <div key={i} data-aos="zoom-in" data-aos-delay={i * 150} className="bg-navy-700 rounded-2xl p-8">
                <div className="w-24 h-24 bg-gradient-to-br from-gold to-orange rounded-full mx-auto mb-4 flex items-center justify-center text-4xl">
                  👤
                </div>
                <h3 className="text-2xl font-bold mb-1">{p.name}</h3>
                <p className="text-gold mb-2">{p.role}</p>
                <p className="text-gray-400 text-sm">{p.exp}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA SECTION */}
      <section className="py-20 px-4">
        <div className="max-w-3xl mx-auto text-center" data-aos="fade-up">
          <h2 className="text-3xl md:text-5xl font-display font-bold mb-4">
            Sẵn sàng bắt đầu hành trình đầu tư?
          </h2>
          <p className="text-gray-300 mb-8 text-lg">
            Đăng ký ngay để giữ vé Webinar Zoom miễn phí + nhận Bộ 10 Ebook Thực Chiến
          </p>
          <a
            href="#lead-form"
            className="inline-block bg-gradient-to-r from-gold to-orange text-navy-900 font-bold text-xl px-12 py-5 rounded-full hover:scale-105 transition-transform shadow-2xl animate-pulse-glow"
          >
            🚀 ĐĂNG KÝ NGAY - MIỄN PHÍ
          </a>
        </div>
      </section>

      {/* FOOTER */}
      <footer className="bg-navy-900 border-t border-gray-800 py-8 px-4 text-center text-gray-500 text-sm">
        <p>© 2026 APEX Fintech. All rights reserved.</p>
        <p className="mt-2">Đầu tư hàng hóa phái sinh có rủi ro. Vui lòng tìm hiểu kỹ trước khi tham gia.</p>
      </footer>
    </main>
  );
}
