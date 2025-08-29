import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthService } from '@/lib/api'

export default function CallbackHandler() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [error, setError] = useState<string>('')
  const router = useRouter()

  useEffect(() => {
    // Debug imediato
    console.log('=== CALLBACK HANDLER MOUNTED ===')
    alert('CallbackHandler carregou!')

    const handleCallback = async () => {
      try {
        // Pega os parâmetros diretamente da URL
        const urlParams = new URLSearchParams(window.location.search)
        const code = urlParams.get('code')
        const urlError = urlParams.get('error')
        
        console.log('=== CALLBACK DEBUG ===')
        console.log('Current URL:', window.location.href)
        console.log('Code:', code)
        console.log('Error:', urlError)
        
        if (urlError) {
          setStatus('error')
          setError(`Acesso negado: ${urlError}`)
          return
        }

        if (!code) {
          setStatus('error')
          setError('Código de autorização não encontrado')
          return
        }

        console.log('Fazendo chamada para API...')
        
        // Trocar o code pelos tokens através da API
        const response = await AuthService.handleCallback(code)
        console.log('Resposta da API:', response)
        
        setStatus('success')

        setTimeout(() => {
          router.push('/dashboard')
        }, 2000)
      } catch (err: any) {
        console.error('Erro:', err)
        setStatus('error')
        setError(err?.response?.data?.error || err?.message || 'Falha na autenticação')
      }
    }

    handleCallback()
  }, [router])

  return (
    <div className="min-h-screen bg-black text-white flex items-center justify-center">
      <div className="text-center">
        {status === 'loading' && (
          <>
            <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-green-500 mx-auto mb-4"></div>
            <h2 className="text-2xl font-semibold mb-2">Conectando com Spotify...</h2>
            <p className="text-gray-400">Aguarde enquanto processamos sua autenticação</p>
          </>
        )}

        {status === 'success' && (
          <>
            <div className="h-16 w-16 bg-green-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <span className="text-2xl">✓</span>
            </div>
            <h2 className="text-2xl font-semibold mb-2 text-green-500">Conectado com sucesso!</h2>
            <p className="text-gray-400">Redirecionando para o dashboard...</p>
          </>
        )}

        {status === 'error' && (
          <>
            <div className="h-16 w-16 bg-red-500 rounded-full flex items-center justify-center mx-auto mb-4">
              <span className="text-2xl">✗</span>
            </div>
            <h2 className="text-2xl font-semibold mb-2 text-red-500">Erro na conexão</h2>
            <p className="text-gray-400 mb-4">{error}</p>
            <button
              onClick={() => router.push('/')}
              className="bg-green-500 text-white px-6 py-2 rounded-lg hover:bg-green-600"
            >
              Tentar novamente
            </button>
          </>
        )}
      </div>
    </div>
  )
}