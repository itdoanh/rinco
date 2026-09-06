/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.ADMIN_API_URL || 'http://localhost:8081'}/:path*`,
      },
    ];
  },
};

export default nextConfig;
