-- Migration script para adicionar novos campos à tabela listening_history
-- Data: 2025-08-28

-- Adicionar novos campos à tabela listening_history
ALTER TABLE listening_history 
ADD COLUMN IF NOT EXISTS listened_duration_ms INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS listening_percentage DECIMAL(5,2) DEFAULT 0,
ADD COLUMN IF NOT EXISTS platform VARCHAR(20),
ADD COLUMN IF NOT EXISTS country VARCHAR(10),
ADD COLUMN IF NOT EXISTS shuffle BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS skipped BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS offline BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS incognito_mode BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS reason_start VARCHAR(50),
ADD COLUMN IF NOT EXISTS reason_end VARCHAR(50);

-- Criar índices para melhorar performance nas novas colunas mais consultadas
CREATE INDEX IF NOT EXISTS idx_listening_history_listened_duration ON listening_history(listened_duration_ms);
CREATE INDEX IF NOT EXISTS idx_listening_history_platform ON listening_history(platform);
CREATE INDEX IF NOT EXISTS idx_listening_history_country ON listening_history(country);

-- Comentários para documentar os novos campos
COMMENT ON COLUMN listening_history.listened_duration_ms IS 'Tempo real escutado em milissegundos (ms_played do Spotify)';
COMMENT ON COLUMN listening_history.listening_percentage IS 'Porcentagem da música que foi escutada';
COMMENT ON COLUMN listening_history.platform IS 'Plataforma utilizada (ios, android, web, etc.)';
COMMENT ON COLUMN listening_history.country IS 'País de origem da escuta';
COMMENT ON COLUMN listening_history.shuffle IS 'Indica se o shuffle estava ativo';
COMMENT ON COLUMN listening_history.skipped IS 'Indica se a música foi pulada';
COMMENT ON COLUMN listening_history.offline IS 'Indica se estava em modo offline';
COMMENT ON COLUMN listening_history.incognito_mode IS 'Indica se estava em modo incógnito';
COMMENT ON COLUMN listening_history.reason_start IS 'Motivo do início da reprodução';
COMMENT ON COLUMN listening_history.reason_end IS 'Motivo do fim da reprodução';