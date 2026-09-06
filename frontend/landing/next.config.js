/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8085',
    NEXT_PUBLIC_LANDING_SERVICE_URL: process.env.NEXT_PUBLIC_LANDING_SERVICE_URL || 'http://localhost:8085',
  },
};

module.exports = nextConfig;
