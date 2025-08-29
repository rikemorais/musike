/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    domains: ['i.scdn.co', 'mosaic.scdn.co', 'lineup-images.scdn.co'],
  },
  output: 'standalone',
  // Configuração para HTTPS em desenvolvimento
  experimental: {
    // removido https: true para evitar problemas no Docker
  },
}

module.exports = nextConfig
