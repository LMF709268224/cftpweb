const isBuild = process.env.NODE_ENV === "production"

/** @type {import('next').NextConfig} */
const nextConfig = {
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  output: isBuild ? "export" : undefined,
  distDir: "build",
  ...(isBuild
    ? {}
    : {
        async rewrites() {
          return [
            {
              source: "/api/:path*",
              destination: "http://localhost:8080/api/:path*",
            },
          ]
        },
      }),
}

export default nextConfig
