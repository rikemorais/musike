'use client'

import { useEffect, useState } from 'react'
import dynamic from 'next/dynamic'

// Carrega o componente dinamicamente, só no cliente
const CallbackHandler = dynamic(() => import('./CallbackHandler'), {
  ssr: false,
  loading: () => <div className="min-h-screen bg-black text-white flex items-center justify-center">Carregando...</div>
})

export default function CallbackPage() {
  return <CallbackHandler />
}
