import LeadForm from '@/components/LeadForm';

export default function HomePage() {
  return (
    <div>
      {/* Hero Section */}
      <section className="relative overflow-hidden bg-gradient-to-br from-cyan-50 via-white to-blue-50 py-20 lg:py-32">
        <div className="absolute inset-0 bg-grid-pattern opacity-5"></div>
        <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
            <div className="animate-fade-in">
              <div className="inline-block px-4 py-2 bg-cyan-100 text-cyan-700 rounded-full text-sm font-medium mb-6">
                CRM thông minh cho doanh nghiệp Việt
              </div>
              <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold text-gray-900 mb-6 leading-tight">
                Quản lý khách hàng
                <span className="block bg-gradient-to-r from-cyan-500 to-blue-600 bg-clip-text text-transparent">
                  thông minh hơn với AI
                </span>
              </h1>
              <p className="text-lg text-gray-600 mb-8 max-w-lg">
                RINCO giúp doanh nghiệp tăng 3x tỷ lệ chuyển đổi, giảm 60% thời gian nhập liệu với
                AI tự động, tích hợp đa kênh và bảo mật zero-trust.
              </p>
              <div className="flex flex-wrap gap-4">
                <a
                  href="#contact"
                  className="px-8 py-4 bg-gradient-to-r from-cyan-500 to-blue-600 text-white font-semibold rounded-lg shadow-lg hover:shadow-xl transform hover:-translate-y-0.5 transition-all duration-200"
                >
                  Dùng thử miễn phí
                </a>
                <a
                  href="#features"
                  className="px-8 py-4 bg-white text-gray-700 font-semibold rounded-lg border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200"
                >
                  Xem tính năng
                </a>
              </div>

              <div className="mt-12 grid grid-cols-3 gap-6">
                <div>
                  <div className="text-3xl font-bold text-gray-900">1000+</div>
                  <div className="text-sm text-gray-600">Doanh nghiệp</div>
                </div>
                <div>
                  <div className="text-3xl font-bold text-gray-900">50M+</div>
                  <div className="text-sm text-gray-600">Khách hàng</div>
                </div>
                <div>
                  <div className="text-3xl font-bold text-gray-900">99.99%</div>
                  <div className="text-sm text-gray-600">Uptime</div>
                </div>
              </div>
            </div>

            <div className="relative animate-slide-up">
              <div className="absolute -inset-4 bg-gradient-to-r from-cyan-400 to-blue-500 rounded-3xl blur-3xl opacity-20"></div>
              <div className="relative bg-white rounded-2xl shadow-2xl p-6 border border-gray-100">
                <div className="flex items-center space-x-2 mb-4">
                  <div className="w-3 h-3 rounded-full bg-red-400"></div>
                  <div className="w-3 h-3 rounded-full bg-yellow-400"></div>
                  <div className="w-3 h-3 rounded-full bg-green-400"></div>
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between p-3 bg-cyan-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-cyan-500 rounded-full flex items-center justify-center text-white font-bold">
                        A
                      </div>
                      <div>
                        <div className="font-medium text-gray-900">Anh Nguyễn Văn A</div>
                        <div className="text-sm text-gray-500">vừa đăng ký dùng thử</div>
                      </div>
                    </div>
                    <div className="text-xs text-gray-400">2 phút trước</div>
                  </div>
                  <div className="flex items-center justify-between p-3 bg-blue-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-blue-500 rounded-full flex items-center justify-center text-white font-bold">
                        B
                      </div>
                      <div>
                        <div className="font-medium text-gray-900">Chị Trần Thị B</div>
                        <div className="text-sm text-gray-500">đã tạo hợp đồng mới</div>
                      </div>
                    </div>
                    <div className="text-xs text-gray-400">5 phút trước</div>
                  </div>
                  <div className="flex items-center justify-between p-3 bg-green-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-green-500 rounded-full flex items-center justify-center text-white font-bold">
                        C
                      </div>
                      <div>
                        <div className="font-medium text-gray-900">Anh Lê Văn C</div>
                        <div className="text-sm text-gray-500">AI đã chấm điểm lead: 92/100</div>
                      </div>
                    </div>
                    <div className="text-xs text-gray-400">8 phút trước</div>
                  </div>
                  <div className="flex items-center justify-between p-3 bg-purple-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className="w-10 h-10 bg-purple-500 rounded-full flex items-center justify-center text-white font-bold">
                        D
                      </div>
                      <div>
                        <div className="font-medium text-gray-900">Chị Phạm Thị D</div>
                        <div className="text-sm text-gray-500">đã hoàn thành onboarding</div>
                      </div>
                    </div>
                    <div className="text-xs text-gray-400">12 phút trước</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 lg:py-32 bg-white">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto mb-16">
            <div className="inline-block px-4 py-2 bg-cyan-100 text-cyan-700 rounded-full text-sm font-medium mb-4">
              Tính năng nổi bật
            </div>
            <h2 className="text-3xl lg:text-5xl font-bold text-gray-900 mb-4">
              Mọi thứ bạn cần để phát triển kinh doanh
            </h2>
            <p className="text-lg text-gray-600">
              Từ quản lý khách hàng, tự động hóa marketing đến phân tích AI - tất cả trong một nền tảng
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {[
              {
                icon: '🤖',
                title: 'AI Lead Scoring',
                description: 'Tự động chấm điểm lead bằng machine learning, ưu tiên lead chất lượng cao',
              },
              {
                icon: '📊',
                title: 'Real-time Analytics',
                description: 'Dashboard phân tích thời gian thực với dữ liệu từ mọi kênh',
              },
              {
                icon: '🔄',
                title: 'Workflow Automation',
                description: 'Tự động hóa quy trình bán hàng, marketing và chăm sóc khách hàng',
              },
              {
                icon: '💬',
                title: 'Multi-channel',
                description: 'Email, SMS, Zalo, Facebook - tất cả trong một hộp thư thống nhất',
              },
              {
                icon: '🔒',
                title: 'Bảo mật Zero-trust',
                description: 'FIDO2, PASETO, Argon2 PoW - bảo mật cấp doanh nghiệp',
              },
              {
                icon: '⚡',
                title: 'Hiệu năng cao',
                description: 'Microservices với Rust/Go, latency < 50ms cho mọi tác vụ',
              },
            ].map((feature, idx) => (
              <div
                key={idx}
                className="group p-6 bg-white border border-gray-100 rounded-2xl hover:border-cyan-200 hover:shadow-lg transition-all duration-200"
              >
                <div className="text-4xl mb-4">{feature.icon}</div>
                <h3 className="text-xl font-semibold text-gray-900 mb-2">{feature.title}</h3>
                <p className="text-gray-600">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Contact Section */}
      <section id="contact" className="py-20 lg:py-32 bg-gray-50">
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-block px-4 py-2 bg-cyan-100 text-cyan-700 rounded-full text-sm font-medium mb-4">
                Liên hệ với chúng tôi
              </div>
              <h2 className="text-3xl lg:text-5xl font-bold text-gray-900 mb-4">
                Sẵn sàng bắt đầu?
              </h2>
              <p className="text-lg text-gray-600 mb-8">
                Đăng ký để nhận 14 ngày dùng thử miễn phí. Đội ngũ của chúng tôi sẽ hỗ trợ bạn
                triển khai và tối ưu cho doanh nghiệp của bạn.
              </p>

              <div className="space-y-4">
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-cyan-100 rounded-lg flex items-center justify-center">
                    <svg className="w-5 h-5 text-cyan-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <div>
                    <div className="text-sm text-gray-500">Email</div>
                    <div className="font-medium text-gray-900">contact@rinco.vn</div>
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-cyan-100 rounded-lg flex items-center justify-center">
                    <svg className="w-5 h-5 text-cyan-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                    </svg>
                  </div>
                  <div>
                    <div className="text-sm text-gray-500">Hotline</div>
                    <div className="font-medium text-gray-900">1900 6868</div>
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-cyan-100 rounded-lg flex items-center justify-center">
                    <svg className="w-5 h-5 text-cyan-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </div>
                  <div>
                    <div className="text-sm text-gray-500">Địa chỉ</div>
                    <div className="font-medium text-gray-900">Hà Nội, Việt Nam</div>
                  </div>
                </div>
              </div>
            </div>

            <div>
              <LeadForm />
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
