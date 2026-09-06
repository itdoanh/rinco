/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Bypass CSP issues for now (chiase_cu uses inline scripts)
  experimental: {
    optimizePackageImports: ['framer-motion', 'aos'],
  },
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: `${process.env.LANDING_API_URL || 'http://localhost:8086'}/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
