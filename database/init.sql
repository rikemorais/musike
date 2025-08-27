-- Criar extensões necessárias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela de usuários
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    spotify_id VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    email VARCHAR(255),
    country VARCHAR(10),
    followers_count INTEGER DEFAULT 0,
    profile_image_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de tokens do Spotify
CREATE TABLE spotify_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de artistas
CREATE TABLE artists (
    id VARCHAR(255) PRIMARY KEY, -- Spotify Artist ID
    name VARCHAR(255) NOT NULL,
    genres TEXT[], -- Array de gêneros
    popularity INTEGER,
    image_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de álbuns
CREATE TABLE albums (
    id VARCHAR(255) PRIMARY KEY, -- Spotify Album ID
    name VARCHAR(255) NOT NULL,
    release_date DATE,
    image_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de faixas
CREATE TABLE tracks (
    id VARCHAR(255) PRIMARY KEY, -- Spotify Track ID
    name VARCHAR(255) NOT NULL,
    album_id VARCHAR(255) REFERENCES albums(id),
    duration_ms INTEGER,
    popularity INTEGER,
    preview_url TEXT,
    isrc VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de relação artista-faixa (muitos para muitos)
CREATE TABLE track_artists (
    track_id VARCHAR(255) REFERENCES tracks(id) ON DELETE CASCADE,
    artist_id VARCHAR(255) REFERENCES artists(id) ON DELETE CASCADE,
    PRIMARY KEY (track_id, artist_id)
);

-- Tabela de histórico de escuta
CREATE TABLE listening_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    track_id VARCHAR(255) REFERENCES tracks(id),
    played_at TIMESTAMP NOT NULL,
    context_type VARCHAR(50), -- playlist, album, artist, etc.
    context_uri VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de analytics pré-computados
CREATE TABLE user_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    time_range VARCHAR(20) NOT NULL, -- short_term, medium_term, long_term
    total_listening_time_ms BIGINT DEFAULT 0,
    diversity_score DECIMAL(5,2) DEFAULT 0,
    top_genres JSONB,
    listening_patterns JSONB,
    monthly_stats JSONB,
    computed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, time_range)
);

-- Índices para performance
CREATE INDEX idx_users_spotify_id ON users(spotify_id);
CREATE INDEX idx_spotify_tokens_user_id ON spotify_tokens(user_id);
CREATE INDEX idx_listening_history_user_id ON listening_history(user_id);
CREATE INDEX idx_listening_history_played_at ON listening_history(played_at);
CREATE INDEX idx_user_analytics_user_id ON user_analytics(user_id);

-- Função para atualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers para atualizar updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_spotify_tokens_updated_at BEFORE UPDATE ON spotify_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
