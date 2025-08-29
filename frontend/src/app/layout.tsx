import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Musike - Spotify Analytics',
  description: 'Analise suas estatísticas de música do Spotify com insights detalhados',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="pt-BR">
      <body className="bg-spotify-black text-white min-h-screen font-sans">
        {children}
      </body>
    </html>
  )
}
