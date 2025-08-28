-- Script para limpar e recriar as tabelas com novos campos
-- CUIDADO: Isso vai apagar TODOS os dados existentes!

-- Remover dados das tabelas na ordem correta (devido às foreign keys)
TRUNCATE TABLE listening_history CASCADE;
TRUNCATE TABLE user_analytics CASCADE;
TRUNCATE TABLE track_artists CASCADE;
TRUNCATE TABLE tracks CASCADE;
TRUNCATE TABLE albums CASCADE;
TRUNCATE TABLE artists CASCADE;
TRUNCATE TABLE spotify_tokens CASCADE;
-- Não truncamos users pois pode ter dados de autenticação importantes

-- Confirmar que as tabelas estão vazias
SELECT 'listening_history' as table_name, count(*) as count FROM listening_history
UNION ALL
SELECT 'user_analytics', count(*) FROM user_analytics
UNION ALL
SELECT 'track_artists', count(*) FROM track_artists
UNION ALL
SELECT 'tracks', count(*) FROM tracks
UNION ALL
SELECT 'albums', count(*) FROM albums
UNION ALL
SELECT 'artists', count(*) FROM artists;