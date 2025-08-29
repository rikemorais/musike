'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthService } from '@/lib/api'

export default function CallbackClient() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [error, setError] = useState<string>('')
  const router = useRouter()

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // Pega os parâmetros diretamente da URL
        const urlParams = new URLSearchParams(window.location.search)
        const code = urlParams.get('code')
        const urlError = urlParams.get('error')
        
        console.log('=== CALLBACK DEBUG ===')
        console.log('Current URL:', window.location.href)
        console.log('URL Search:', window.location.search)
        console.log('Code:', code)
        console.log('Error:', urlError)
        console.log('API URL:', process.env.NEXT_PUBLIC_API_URL)
        
        if (urlError) {
          console.log('OAuth error detected:', urlError)
          setStatus('error')
          setError(`Acesso negado: ${urlError}`)
          return
        }

        if (!code) {
          console.log('No authorization code found')
          setStatus('error')
          setError('Código de autorização não encontrado')
          return
        }

        console.log('Processing authorization code:', code)
        
        // Trocar o code pelos tokens através da API
        console.log('Calling AuthService.handleCallback...')
        const response = await AuthService.handleCallback(code)
        console.log('Callback response:', response)
        
        setStatus('success')
        console.log('Authentication successful, redirecting in 2 seconds...')

        setTimeout(() => {
          console.log('Redirecting to dashboard...')
          router.push('/dashboard')
        }, 2000)
      } catch (err: any) {
        setStatus('error')
        console.error('=== AUTH ERROR ===')
        console.error('Error object:', err)
        console.error('Error message:', err?.message)
        console.error('Error response:', err?.response)
        console.error('Error response data:', err?.response?.data)
        
        const errorMsg = err?.response?.data?.error || err?.message || 'Falha na autenticação'
        setError(errorMsg)
      }
    }

    handleCallback()
  }, [router])

  return (
    <div className="min-h-screen bg-spotify-black flex items-center justify-center">
      <div className="text-center">
        {status === 'loading' && (
          <>
            <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-spotify-green mx-auto mb-4"></div>
            <h2 className="text-2xl font-semibold mb-2">Conectando com Spotify...</h2>
            <p className="text-gray-400">Aguarde enquanto processamos sua autenticação</p>
          </>
        )}

        {status === 'success' && (
          <>
            <div className="h-16 w-16 bg-spotify-green rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="h-8 w-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h2 className="text-2xl font-semibold mb-2 text-spotify-green">Conectado com sucesso!</h2>
            <p className="text-gray-400">Redirecionando para o dashboard...</p>
          </>
        )}

        {status === 'error' && (
          <>
            <div className="h-16 w-16 bg-red-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="h-8 w-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>
            <h2 className="text-2xl font-semibold mb-2 text-red-500">Erro na conexão</h2>
            <p className="text-gray-400 mb-4">{error}</p>
            <button
              onClick={() => router.push('/')}
              className="spotify-gradient text-white px-6 py-2 rounded-lg hover:scale-105 transition-transform"
            >
              Tentar novamente
            </button>
          </>
        )}
      </div>
    </div>
  )
}