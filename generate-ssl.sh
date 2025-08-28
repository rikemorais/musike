#!/bin/bash

# Script para gerar certificados SSL auto-assinados para desenvolvimento local
echo "🔐 Gerando certificados SSL para desenvolvimento local..."

# Criar diretório para certificados
mkdir -p certs

# Gerar certificado auto-assinado
openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes -subj "/C=BR/ST=SP/L=SaoPaulo/O=Musike/OU=Dev/CN=localhost" -addext "subjectAltName=DNS:localhost,DNS:127.0.0.1,IP:127.0.0.1"

echo "✅ Certificados SSL gerados em ./certs/"
echo "📝 Agora você precisa:"
echo "   1. Adicionar o certificado como confiável no macOS"
echo "   2. Configurar as URLs HTTPS no Spotify Dashboard"
echo ""
echo "🔧 Para adicionar como confiável no macOS:"
echo "   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./certs/cert.pem"
echo ""
echo "🌐 URLs para o Spotify Dashboard:"
echo "   https://localhost:3000/callback"
echo "   https://127.0.0.1:3000/callback"
