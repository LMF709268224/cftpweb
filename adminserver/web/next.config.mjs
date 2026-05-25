const isBuild = process.env.NODE_ENV === "production";

/** @type {import('next').NextConfig} */
const nextConfig = {
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  // 关键配置 1: 仅在生产构建时启用静态导出 (开发模式下关闭，以便支持 proxy)
  output: isBuild ? "export" : undefined,
  // 关键配置 2: 把输出目录从默认的 'out' 改为 'build'，对齐我们的 Go 后端和 Dockerfile
  distDir: "build",

  // 关键配置 3: 本地开发时的接口代理 (替代 Vite 的 proxy)
  async rewrites() {
    if (isBuild) return [];
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*", // 转发到你的 Go 后端
      },
    ];
  },
}

export default nextConfig
