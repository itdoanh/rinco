/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  transpilePackages: ['@repo/ui'],
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    NEXT_PUBLIC_SIGNALING_URL: process.env.NEXT_PUBLIC_SIGNALING_URL || 'ws://localhost:8080/ws',
  },
};

export default nextConfig;
