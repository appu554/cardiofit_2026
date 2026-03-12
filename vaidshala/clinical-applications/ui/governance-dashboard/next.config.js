/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,

  // Environment variables for KB-0 API
  env: {
    KB0_API_URL: process.env.KB0_API_URL || 'http://localhost:8080',
  },

  // Enable standalone output for Docker deployment (not used on Vercel)
  ...(process.env.VERCEL ? {} : { output: 'standalone' }),

  // API rewrites to proxy KB-0 API calls (avoids CORS)
  async rewrites() {
    return [
      {
        source: '/api/v2/:path*',
        destination: `${process.env.KB0_API_URL || 'http://localhost:8080'}/api/v2/:path*`,
      },
    ];
  },

  // Webpack: fix pdfjs-dist .mjs bundling with Next.js webpack 5
  webpack: (config) => {
    config.resolve.alias.canvas = false;

    // pdfjs-dist ships .mjs files containing an internal webpack runtime.
    // Next.js's built-in oneOf rules treat .mjs as strict ESM, which
    // collides with that internal runtime → "Object.defineProperty called
    // on non-object". Fix: inject a higher-priority rule that processes
    // pdfjs-dist .mjs as 'javascript/auto' BEFORE Next.js oneOf rules.
    config.module.rules.unshift({
      test: /\.mjs$/,
      include: /node_modules[\\/]pdfjs-dist/,
      type: 'javascript/auto',
      resolve: { fullySpecified: false },
    });

    return config;
  },
};

module.exports = nextConfig;
