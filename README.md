# Musike - Plataforma de Analytics do Spotify

## Stack Tecnológica

### Backend
- **Go** com Gin Framework
- **PostgreSQL** para dados relacionais
- **Redis** para cache e tokens
- **OAuth 2.0** integração com Spotify API

### Frontend
- **Next.js 14** com App Router
- **TypeScript** para type safety
- **Tailwind CSS** para styling
- **Recharts** para visualizações
- **Framer Motion** para animações

## Configuração Rápida

### 1. Configurar Spotify App
1. Acesse https://developer.spotify.com/dashboard
2. Crie uma nova aplicação
3. Configure o Redirect URI: `http://localhost:3000/callback`
4. Anote Client ID e Client Secret

### 2. Configurar Variáveis de Ambiente
```bash
# Backend (.env na pasta backend/)
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
SPOTIFY_REDIRECT_URL=http://localhost:3000/callback
JWT_SECRET=your-super-secret-jwt-key
DATABASE_URL=postgres://musike_user:musike_password@localhost:5432/musike?sslmode=disable
REDIS_URL=redis://localhost:6379
PORT=8080

# Frontend (.env.local na pasta frontend/)
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 3. Executar com Docker
```bash
# Executar toda a stack
docker-compose up --build

# Ou executar individualmente
docker-compose up postgres redis  # Bancos de dados
docker-compose up backend         # API Go
docker-compose up frontend        # Next.js
```

### 4. Executar em Desenvolvimento
```bash
# Backend
cd backend
go run main.go

# Frontend (em outro terminal)
cd frontend
npm run dev
```

## Arquitetura

### APIs Disponíveis
- `GET /api/v1/auth/spotify` - URL de autenticação Spotify
- `GET /api/v1/auth/callback` - Callback OAuth
- `GET /api/v1/user/profile` - Perfil do usuário
- `GET /api/v1/user/top-tracks` - Top músicas
- `GET /api/v1/user/top-artists` - Top artistas
- `GET /api/v1/user/analytics` - Analytics completos
- `GET /api/v1/user/recommendations` - Recomendações

### Funcionalidades

#### Dashboard Analytics
- **Tempo total de escuta** - Métricas agregadas
- **Score de diversidade** - Baseado em gêneros e artistas
- **Gêneros favoritos** - Visualização em pizza
- **Padrões semanais** - Gráfico de barras por dia
- **Atividade recente** - Timeline dos últimos 7 dias
- **Top tracks/artists** - Listas rankiadas

#### Performance
- **Go backend** - Alta concorrência, baixa latência
- **Redis cache** - Cache de tokens e consultas frequentes
- **PostgreSQL** - Dados estruturados com índices otimizados
- **Next.js SSR** - Carregamento rápido de páginas

### Próximos Passos
1. Implementar sistema de jobs para coleta contínua
2. Adicionar ClickHouse para analytics em tempo real
3. Implementar WebSockets para updates live
4. Adicionar testes automatizados
5. Deploy com Kubernetes

## Acesso
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379
