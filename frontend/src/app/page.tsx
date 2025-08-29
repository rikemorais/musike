'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthService } from '@/lib/api'
import { Music, TrendingUp, Clock, Users } from 'lucide-react'

export default function HomePage() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    const checkAuth = () => {
      const authenticated = AuthService.isAuthenticated()
      setIsAuthenticated(authenticated)
      setIsLoading(false)

      if (authenticated) {
        router.push('/dashboard')
      }
    }

    checkAuth()
  }, [router])

  const handleSpotifyLogin = async () => {
    try {
      const { auth_url } = await AuthService.getSpotifyAuthUrl()
      window.location.href = auth_url
    } catch (error) {
      console.error('Erro ao obter URL de autenticacao:', error)
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-green-400"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 text-white">
      <header className="container mx-auto px-6 py-8">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Music className="h-8 w-8 text-green-400" />
            <h1 className="text-2xl font-bold">Musike</h1>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-6 py-12">
        <div className="text-center max-w-4xl mx-auto">
          <h2 className="text-5xl font-bold mb-6 bg-gradient-to-r from-green-400 to-green-600 bg-clip-text text-transparent">
            Descubra Suas Estatisticas de Musica
          </h2>
          <p className="text-xl text-gray-300 mb-12 leading-relaxed">
            Conecte-se ao Spotify e explore insights detalhados sobre seus habitos musicais,
            artistas favoritos, generos preferidos e muito mais.
          </p>

          <div className="grid md:grid-cols-3 gap-8 mb-12">
            <div className="bg-gray-800 bg-opacity-50 backdrop-blur-sm rounded-lg p-6 border border-gray-700">
              <div className="h-12 w-12 bg-green-400 rounded-lg mx-auto mb-4 flex items-center justify-center">
                <TrendingUp className="h-6 w-6 text-gray-900" />
              </div>
              <h3 className="text-xl font-semibold mb-2">Analytics Avancados</h3>
              <p className="text-gray-400">
                Visualize suas tendencias musicais com graficos interativos e metricas detalhadas
              </p>
            </div>
            <div className="bg-gray-800 bg-opacity-50 backdrop-blur-sm rounded-lg p-6 border border-gray-700">
              <div className="h-12 w-12 bg-green-400 rounded-lg mx-auto mb-4 flex items-center justify-center">
                <Users className="h-6 w-6 text-gray-900" />
              </div>
              <h3 className="text-xl font-semibold mb-2">Top Artistas</h3>
              <p className="text-gray-400">
                Descubra seus artistas mais ouvidos em diferentes periodos de tempo
              </p>
            </div>
            <div className="bg-gray-800 bg-opacity-50 backdrop-blur-sm rounded-lg p-6 border border-gray-700">
              <div className="h-12 w-12 bg-green-400 rounded-lg mx-auto mb-4 flex items-center justify-center">
                <Clock className="h-6 w-6 text-gray-900" />
              </div>
              <h3 className="text-xl font-semibold mb-2">Padroes de Escuta</h3>
              <p className="text-gray-400">
                Analise quando e como voce mais escuta musica durante o dia
              </p>
            </div>
          </div>

          <button
            onClick={handleSpotifyLogin}
            className="bg-gradient-to-r from-green-400 to-green-600 text-white px-8 py-4 rounded-full text-lg font-semibold hover:scale-105 transition-transform duration-200 shadow-lg"
          >
            Conectar com Spotify
          </button>
        </div>
      </main>

      <footer className="container mx-auto px-6 py-8 text-center text-gray-400">
        <p>2025 Musike. Feito com amor para amantes de musica.</p>
      </footer>
    </div>
  )
}