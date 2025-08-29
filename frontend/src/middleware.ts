import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Se a rota for /callback com parâmetros de query, redireciona para auth-callback.html
  if (request.nextUrl.pathname === '/callback' && request.nextUrl.searchParams.has('code')) {
    const url = new URL('/auth-callback.html', request.url)
    url.search = request.nextUrl.search // Preserva os parâmetros de query
    return NextResponse.redirect(url)
  }
}

export const config = {
  matcher: '/callback'
}