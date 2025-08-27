# Guia de Configuração do Spotify - Musike

## Problema: "This redirect URI is not secure"

Este erro acontece quando o Spotify não reconhece o redirect URI como válido. Aqui está a solução completa:

## 1. Configuração no Spotify Developer Dashboard

### Passo 1: Acesse o Dashboard
1. Vá para https://developer.spotify.com/dashboard
2. Faça login com sua conta Spotify
3. Clique em "Create App"

### Passo 2: Criar a Aplicação
- **App name**: Musike Analytics
- **App description**: Plataforma de analytics para estatísticas do Spotify
- **Website**: http://localhost:3000 (para desenvolvimento)
- **Redirect URIs**: 
  - `https://localhost:3000/callback`
  - `https://127.0.0.1:3000/callback`
- **APIs used**: Web API
- **Commercial/Non-commercial**: Non-commercial (para desenvolvimento)

### Passo 3: Configurar Redirect URIs
⚠️ **IMPORTANTE**: No campo "Redirect URIs", adicione EXATAMENTE estas URLs:

```
https://localhost:3000/callback
https://127.0.0.1:3000/callback
```

### Passo 4: Obter Credenciais
Após criar a app:
1. Clique em "Settings"
2. Copie o **Client ID**
3. Clique em "View client secret" e copie o **Client Secret**

## 2. Configuração Local

### Criar arquivo .env no backend
Crie o arquivo `backend/.env` com suas credenciais:

```bash
# Configurações do Spotify (SUBSTITUA com suas credenciais reais)
SPOTIFY_CLIENT_ID=seu_client_id_aqui
SPOTIFY_CLIENT_SECRET=seu_client_secret_aqui
SPOTIFY_REDIRECT_URL=http://localhost:3000/callback

# JWT Secret (gere uma chave segura)
JWT_SECRET=sua-chave-jwt-super-secreta-aqui

# Database
DATABASE_URL=postgres://musike_user:musike_password@localhost:5432/musike?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# Server
PORT=8080
```

### Criar arquivo .env.local no frontend
Crie o arquivo `frontend/.env.local`:

```bash
NEXT_PUBLIC_API_URL=https://localhost:8080
```

## 3. Verificações Importantes

### URLs Permitidas pelo Spotify:
✅ `https://localhost:3000/callback`
✅ `https://127.0.0.1:3000/callback`
❌ URLs com porta diferente sem configurar

### Scopes Necessários:
A aplicação usa estes scopes (já configurados no código):
- `user-read-private`
- `user-read-email`
- `user-top-read`
- `user-read-recently-played`
- `user-library-read`
- `playlist-read-private`
- `user-read-playback-state`
- `user-read-currently-playing`

## 4. Como Testar

1. **Executar a aplicação**:
   ```bash
   # Terminal 1: Backend
   cd backend
   go run main.go

   # Terminal 2: Frontend
   cd frontend
   npm run dev
   ```

2. **Acessar**: https://localhost:3000
3. **Clicar em "Conectar com Spotify"**
4. **Autorizar a aplicação** no popup do Spotify
5. **Você será redirecionado** para o dashboard com seus dados

## 5. Troubleshooting

### Se ainda der erro:
1. **Verifique se as URLs estão exatas** no Spotify Dashboard
2. **Confirme que está usando http://** (não https) para localhost
3. **Teste com 127.0.0.1** se localhost não funcionar
4. **Limpe cache do navegador** e cookies do Spotify
5. **Verifique se o backend está rodando** na porta 8080

### Logs úteis:
- Backend: Verifique logs no terminal onde rodou `go run main.go`
- Frontend: Abra DevTools (F12) e veja erros no Console
- Network: Verifique se as chamadas para `/api/v1/auth/spotify` funcionam

## 6. Para Produção (Futuro)

Quando deployar em produção, você precisará:
1. **Registrar domínio real** no Spotify Dashboard
2. **Usar HTTPS** obrigatoriamente
3. **Atualizar redirect URIs** para URLs de produção
4. **Configurar variáveis de ambiente** no servidor

Exemplo para produção:
```
https://seudominio.com/callback
```
