/** @type {import('next').NextConfig} */
const nextConfig = {
  async headers() {
    return [{ source: "/:path*", headers: [
      {key:"X-Frame-Options",value:"DENY"},
      {key:"X-Content-Type-Options",value:"nosniff"},
      {key:"Referrer-Policy",value:"no-referrer"},
      {key:"Permissions-Policy",value:"microphone=(self), camera=()"},
    ] }];
  },
  async rewrites() {
    const core = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:18080";
    const voice = process.env.VOICE_INTERNAL_URL ?? "http://127.0.0.1:18100";
    return { afterFiles: [
      { source: "/health", destination: `${core}/health` },
      { source: "/api/:path*", destination: `${core}/api/:path*` },
      { source: "/voice/:path*", destination: `${voice}/:path*` },
    ] };
  },
};
export default nextConfig;
