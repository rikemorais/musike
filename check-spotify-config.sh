#!/bin/bash

# Script de Verificação de Configuração do Spotify - Musike
echo "🔍 Verificando configuração do Spotify para Musike..."
echo "=================================================="

# Verificar se arquivos .env existem
echo "📁 Verificando arquivos de configuração..."

if [ ! -f "backend/.env" ]; then
    echo "❌ Arquivo backend/.env não encontrado!"
    echo "   Crie o arquivo com base no backend/.env.example"
    echo ""
else
    echo "✅ Arquivo backend/.env encontrado"
fi

if [ ! -f "frontend/.env.local" ]; then
    echo "❌ Arquivo frontend/.env.local não encontrado!"
    echo "   Crie o arquivo com: NEXT_PUBLIC_API_URL=http://localhost:8080"
    echo ""
else
    echo "✅ Arquivo frontend/.env.local encontrado"
fi

# Verificar variáveis críticas do backend
echo ""
echo "🔧 Verificando variáveis de ambiente do backend..."

if [ -f "backend/.env" ]; then
    source backend/.env

    if [ -z "$SPOTIFY_CLIENT_ID" ]; then
        echo "❌ SPOTIFY_CLIENT_ID não está definido"
    else
        echo "✅ SPOTIFY_CLIENT_ID: ${SPOTIFY_CLIENT_ID:0:8}..."
    fi

    if [ -z "$SPOTIFY_CLIENT_SECRET" ]; then
        echo "❌ SPOTIFY_CLIENT_SECRET não está definido"
    else
        echo "✅ SPOTIFY_CLIENT_SECRET: ${SPOTIFY_CLIENT_SECRET:0:8}..."
    fi

    if [ -z "$SPOTIFY_REDIRECT_URL" ]; then
        echo "❌ SPOTIFY_REDIRECT_URL não está definido"
    else
        echo "✅ SPOTIFY_REDIRECT_URL: $SPOTIFY_REDIRECT_URL"

        # Verificar se é uma URL válida para desenvolvimento
        if [[ "$SPOTIFY_REDIRECT_URL" == "http://localhost:3000/callback" || "$SPOTIFY_REDIRECT_URL" == "http://127.0.0.1:3000/callback" ]]; then
            echo "   ✅ URL válida para desenvolvimento local"
        else
            echo "   ⚠️  URL pode não ser válida para desenvolvimento local"
            echo "       Use: http://localhost:3000/callback"
        fi
    fi
fi

echo ""
echo "🌐 Verificando conectividade..."

# Verificar se as portas estão livres
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "❌ Porta 3000 já está em uso"
else
    echo "✅ Porta 3000 disponível"
fi

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo "❌ Porta 8080 já está em uso"
else
    echo "✅ Porta 8080 disponível"
fi

echo ""
echo "📋 Checklist para resolver 'redirect URI is not secure':"
echo "=================================================="
echo "1. ✅ Acesse https://developer.spotify.com/dashboard"
echo "2. ✅ Crie uma nova aplicação ou edite existente"
echo "3. ✅ Em 'Settings', adicione EXATAMENTE estas URLs em 'Redirect URIs':"
echo "   - http://localhost:3000/callback"
echo "   - http://127.0.0.1:3000/callback"
echo "4. ✅ Copie Client ID e Client Secret para backend/.env"
echo "5. ✅ Salve as configurações no Spotify Dashboard"
echo "6. ✅ Execute o backend: cd backend && go run main.go"
echo "7. ✅ Execute o frontend: cd frontend && npm run dev"
echo "8. ✅ Acesse http://localhost:3000"
echo ""
echo "🚨 IMPORTANTE: Use HTTP (não HTTPS) para desenvolvimento local!"
echo "🚨 IMPORTANTE: As URLs devem ser EXATAMENTE iguais no Spotify Dashboard!"

echo ""
echo "🔗 URLs úteis:"
echo "- Spotify Dashboard: https://developer.spotify.com/dashboard"
echo "- Documentação OAuth: https://developer.spotify.com/documentation/general/guides/authorization/"
echo "- Frontend local: http://localhost:3000"
echo "- Backend API: http://localhost:8080/api/v1"
