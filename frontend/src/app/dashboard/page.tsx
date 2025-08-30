'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { AuthService, SpotifyAPIService } from '@/lib/api'
import { SpotifyUser, SpotifyTrack, SpotifyArtist, UserAnalytics } from '@/types/spotify'
import { Music, TrendingUp, Clock, Users, LogOut, Play } from 'lucide-react'
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, PieChart, Pie, Cell, LineChart, Line } from 'recharts'
import SpotifyDataImport from '@/components/SpotifyDataImport'

export default function DashboardPage() {
  const [user, setUser] = useState<SpotifyUser | null>(null)
  const [topTracks, setTopTracks] = useState<SpotifyTrack[]>([])
  const [topArtists, setTopArtists] = useState<SpotifyArtist[]>([])
  const [recentTracks, setRecentTracks] = useState<Array<{ track: SpotifyTrack, played_at: string }>>([])
  const [analytics, setAnalytics] = useState<UserAnalytics | null>(null)
  const [timeRange, setTimeRange] = useState('all')
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    if (!AuthService.isAuthenticated()) {
      router.push('/')
      return
    }

    loadDashboardData()
  }, [router, timeRange])

  // Mapear nossos filtros para os filtros do Spotify e analytics
  const getSpotifyTimeRange = (filter: string) => {
    switch (filter) {
      case 'day':
      case 'week':
        return 'short_term' // ~4 semanas
      case 'month':
      case 'quarter':
        return 'medium_term' // ~6 meses  
      case 'semester':
      case 'year':
      case 'all':
        return 'long_term' // vários anos
      default:
        return 'medium_term'
    }
  }

  const getAnalyticsFilter = (filter: string) => {
    switch (filter) {
      case 'day':
        return 'day'
      case 'week':
        return 'week'
      case 'month':
        return 'month'
      case 'quarter':
        return 'quarter'
      case 'semester':
        return 'semester'
      case 'year':
        return 'year'
      case 'all':
        return 'alltime'
      default:
        return 'month'
    }
  }

  const getActivityPeriodLabel = (filter: string) => {
    switch (filter) {
      case 'day':
        return 'Últimas 24 Horas'
      case 'week':
        return 'Últimos 7 Dias'
      case 'month':
        return 'Últimos 30 Dias'
      case 'quarter':
        return 'Últimos 90 Dias'
      case 'semester':
        return 'Últimos 180 Dias'
      case 'year':
        return 'Últimos 365 Dias'
      case 'all':
        return 'Todo o Histórico'
      default:
        return 'Atividade Recente'
    }
  }

  const getActivityDays = (filter: string) => {
    switch (filter) {
      case 'day':
        return 1
      case 'week':
        return 7
      case 'month':
        return 30
      case 'quarter':
        return 90
      case 'semester':
        return 180
      case 'year':
        return 365
      case 'all':
        return 365 // máximo para não sobrecarregar
      default:
        return 30
    }
  }

  const loadDashboardData = async () => {
    try {
      setIsLoading(true)

      const spotifyTimeRange = getSpotifyTimeRange(timeRange)
      const analyticsFilter = getAnalyticsFilter(timeRange)

      const [userProfile, tracksData, artistsData, recentData, analyticsData] = await Promise.all([
        SpotifyAPIService.getUserProfile(),
        SpotifyAPIService.getTopTracks(spotifyTimeRange, 10),
        SpotifyAPIService.getTopArtists(spotifyTimeRange, 10),
        SpotifyAPIService.getRecentlyPlayed(5),
        SpotifyAPIService.getUserAnalytics(analyticsFilter)
      ])

      setUser(userProfile)
      setTopTracks(tracksData.items)
      setTopArtists(artistsData.items)
      setRecentTracks(recentData.items || [])
      setAnalytics(analyticsData)
    } catch (error) {
      console.error('Erro ao carregar dados do dashboard:', error)
      AuthService.logout()
      router.push('/')
    } finally {
      setIsLoading(false)
    }
  }

  const handleLogout = () => {
    AuthService.logout()
    router.push('/')
  }

  const formatDuration = (ms: number) => {
    const minutes = Math.floor(ms / 60000)
    const seconds = Math.floor((ms % 60000) / 1000)
    return `${minutes}:${seconds.toString().padStart(2, '0')}`
  }

  const formatListeningTime = (ms: number) => {
    const hours = Math.floor(ms / 3600000)
    const minutes = Math.floor((ms % 3600000) / 60000)
    
    // Format hours with thousands separator if >= 1000
    const formattedHours = hours >= 1000 
      ? hours.toString().replace(/\B(?=(\d{3})+(?!\d))/g, '.') 
      : hours.toString()
    
    return `${formattedHours}h ${minutes}m`
  }

  const formatTimeAgo = (dateString: string) => {
    const now = new Date()
    const played = new Date(dateString)
    const diffMs = now.getTime() - played.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 1) return 'Agora'
    if (diffMins < 60) return `${diffMins}m atrás`
    if (diffHours < 24) return `${diffHours}h atrás`
    return `${diffDays}d atrás`
  }


  if (isLoading) {
    return (
      <div className="min-h-screen bg-spotify-black flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-spotify-green mx-auto mb-4"></div>
          <p className="text-gray-300">Carregando suas estatísticas...</p>
        </div>
      </div>
    )
  }

  const genreData = analytics?.top_genres.slice(0, 10).map(genre => ({
    name: genre.genre,
    value: genre.percentage,
    count: genre.track_count,
    playCount: genre.play_count,
    totalTime: genre.total_time_ms
  })) || []

  const listeningPatternData = analytics?.listening_patterns.weekday_usage.map((usage, index) => ({
    day: ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'][index],
    usage: Math.round(usage)
  })) || []

  const activityDays = getActivityDays(timeRange)
  const activityData = analytics?.recent_activity
    .slice(-activityDays)
    .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
    .map(activity => ({
      date: new Date(activity.date).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' }),
      tracks: activity.track_count,
      minutes: Math.round(activity.duration_ms / 60000),
      formattedTime: formatListeningTime(activity.duration_ms)
    })) || []

  const COLORS = ['#1DB954', '#1ed760', '#17a844', '#14843b', '#0f6b2f', '#0a5025', '#084020', '#06321b', '#042416', '#021611']

  return (
    <div className="min-h-screen bg-spotify-black">
      <header className="border-b border-gray-800 bg-spotify-dark">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Music className="h-8 w-8 text-spotify-green" />
              <h1 className="text-2xl font-bold">Musike</h1>
              {user && (
                <div className="flex items-center space-x-3 ml-8">
                  {user.images[0] && (
                    <img
                      src={user.images[0].url}
                      alt={user.display_name}
                      className="w-8 h-8 rounded-full"
                    />
                  )}
                  <span className="text-gray-300">Olá, {user.display_name}</span>
                </div>
              )}
            </div>

            <div className="flex items-center space-x-4">
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value)}
                className="bg-gray-800 text-white px-3 py-2 rounded-lg border border-gray-600"
              >
                <option value="day">Último Dia</option>
                <option value="week">Última Semana</option>
                <option value="month">Último Mês</option>
                <option value="quarter">Último Trimestre</option>
                <option value="semester">Último Semestre</option>
                <option value="year">Último Ano</option>
                <option value="all">Todo o Histórico</option>
              </select>


              <button
                onClick={handleLogout}
                className="flex items-center space-x-2 text-gray-300 hover:text-white transition-colors"
              >
                <LogOut className="h-5 w-5" />
                <span>Sair</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="container mx-auto px-6 py-8">
        {analytics && (
          <div className="grid grid-cols-1 md:grid-cols-5 gap-6 mb-8">
            <div className="bg-gray-800 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-gray-400 text-sm">Tempo Total</p>
                  <p className="text-2xl font-bold text-spotify-green">
                    {formatListeningTime(analytics.total_listening_time_ms)}
                  </p>
                </div>
                <Clock className="h-8 w-8 text-spotify-green" />
              </div>
            </div>

            <div className="bg-gray-800 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-gray-400 text-sm">Músicas Tocadas</p>
                  <p className="text-2xl font-bold text-spotify-green">
                    {analytics.total_plays.toLocaleString()}
                  </p>
                </div>
                <Play className="h-8 w-8 text-spotify-green" />
              </div>
            </div>

            <div className="bg-gray-800 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-gray-400 text-sm">Tempo Médio</p>
                  <p className="text-2xl font-bold text-spotify-green">
                    {formatDuration(analytics.average_play_time_ms)}
                  </p>
                </div>
                <TrendingUp className="h-8 w-8 text-spotify-green" />
              </div>
            </div>

            <div className="bg-gray-800 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-gray-400 text-sm">Popularidade Média</p>
                  <p className="text-2xl font-bold text-spotify-green">
                    {Math.round(analytics.average_track_popularity)}/100
                  </p>
                </div>
                <TrendingUp className="h-8 w-8 text-spotify-green" />
              </div>
            </div>

            <div className="bg-gray-800 rounded-lg p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-gray-400 text-sm">Seguidores</p>
                  <p className="text-2xl font-bold text-spotify-green">
                    {user?.followers.total.toLocaleString()}
                  </p>
                </div>
                <Users className="h-8 w-8 text-spotify-green" />
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          <div className="bg-gray-800 rounded-lg p-6">
            <h3 className="text-xl font-semibold mb-4">Top 10 Gêneros Favoritos</h3>
            {genreData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={genreData}
                    cx="50%"
                    cy="50%"
                    outerRadius={100}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, value, cx, cy, midAngle, innerRadius, outerRadius }) => {
                      return (
                        <text 
                          x={cx + (outerRadius + 20) * Math.cos(-midAngle * Math.PI / 180)} 
                          y={cy + (outerRadius + 20) * Math.sin(-midAngle * Math.PI / 180)}
                          fill="#ffffff"
                          textAnchor={cx + (outerRadius + 20) * Math.cos(-midAngle * Math.PI / 180) > cx ? 'start' : 'end'} 
                          dominantBaseline="central"
                          fontSize="12px"
                        >
                          {`${name}: ${value.toFixed(1)}%`}
                        </text>
                      );
                    }}
                    labelLine={false}
                  >
                    {genreData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip 
                    content={({ active, payload }) => {
                      if (active && payload && payload.length) {
                        const data = payload[0].payload;
                        return (
                          <div className="bg-gray-900 border border-gray-600 rounded-lg p-3 text-white">
                            <p className="font-semibold">{data.name}</p>
                            <p className="text-sm">Músicas tocadas: {data.playCount?.toLocaleString() || '0'}</p>
                            <p className="text-sm">Tempo total: {data.totalTime ? formatListeningTime(data.totalTime) : '0h 0m'}</p>
                            <p className="text-sm text-gray-400">{data.value?.toFixed(1) || '0'}% do total</p>
                          </div>
                        );
                      }
                      return null;
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-72">
                <p className="text-gray-400">Nenhum dado de gênero disponível</p>
              </div>
            )}
          </div>

          <div className="bg-gray-800 rounded-lg p-6">
            <h3 className="text-xl font-semibold mb-4">Padrões Semanais</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={listeningPatternData}>
                <XAxis dataKey="day" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="usage" fill="#1DB954" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 mb-8">
          <h3 className="text-xl font-semibold mb-4">Atividade - {getActivityPeriodLabel(timeRange)}</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={activityData}>
              <XAxis dataKey="date" />
              <YAxis yAxisId="left" />
              <YAxis yAxisId="right" orientation="right" />
              <Tooltip 
                content={({ active, payload, label }) => {
                  if (active && payload && payload.length) {
                    const data = payload[0].payload;
                    return (
                      <div className="bg-gray-900 border border-gray-600 rounded-lg p-3 text-white">
                        <p className="font-semibold">{label}</p>
                        <p className="text-sm">Músicas: {data.tracks}</p>
                        <p className="text-sm">Tempo: {data.formattedTime}</p>
                      </div>
                    );
                  }
                  return null;
                }}
              />
              <Line yAxisId="left" type="monotone" dataKey="tracks" stroke="#1DB954" name="Músicas" />
              <Line yAxisId="right" type="monotone" dataKey="minutes" stroke="#1ed760" name="Minutos" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {recentTracks.length > 0 && (
          <div className="bg-gray-800 rounded-lg p-6 mb-8">
            <h3 className="text-xl font-semibold mb-4 flex items-center">
              <Clock className="h-6 w-6 text-spotify-green mr-3" />
              Tocadas Recentemente
            </h3>
            <div className="space-y-3">
              {recentTracks.slice(0, 5).map((item, index) => (
                <div key={index} className="flex items-center space-x-4 p-3 rounded-lg bg-gray-700 hover:bg-gray-600 transition-colors">
                  <div className="w-12 h-12 bg-gray-600 rounded flex-shrink-0 overflow-hidden">
                    {item.track?.album?.images && item.track.album.images.length > 0 ? (
                      <img
                        src={item.track.album.images[0].url}
                        alt={item.track.album.name || 'Album cover'}
                        className="w-12 h-12 object-cover"
                        onError={(e) => {
                          e.currentTarget.style.display = 'none';
                          const nextElement = e.currentTarget.nextElementSibling as HTMLElement;
                          if (nextElement) {
                            nextElement.style.display = 'flex';
                          }
                        }}
                      />
                    ) : null}
                    <div className="w-12 h-12 bg-gray-600 rounded flex items-center justify-center" style={{display: item.track?.album?.images?.length > 0 ? 'none' : 'flex'}}>
                      <Music className="h-6 w-6 text-gray-400" />
                    </div>
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{item.track?.name}</p>
                    <p className="text-gray-400 text-sm truncate">
                      {item.track?.artists?.map(artist => artist.name).join(', ')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-gray-400">{formatTimeAgo(item.played_at)}</p>
                    <p className="text-xs text-gray-500">{formatDuration(item.track?.duration_ms || 0)}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <div className="bg-gray-800 rounded-lg p-6">
            <h3 className="text-xl font-semibold mb-4">Suas Músicas Favoritas</h3>
            <div className="space-y-3">
              {topTracks.map((track, index) => (
                <div key={track.id} className="flex items-center space-x-3 p-3 rounded-lg bg-gray-700 hover:bg-gray-600 transition-colors">
                  <span className="text-spotify-green font-bold w-6">{index + 1}</span>
                  {track.album.images[0] && (
                    <img
                      src={track.album.images[0].url}
                      alt={track.album.name}
                      className="w-12 h-12 rounded"
                    />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{track.name}</p>
                    <p className="text-gray-400 text-sm truncate">
                      {track.artists.map(artist => artist.name).join(', ')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-gray-400">{formatDuration(track.duration_ms)}</p>
                    {track.preview_url && (
                      <Play className="h-4 w-4 text-spotify-green mt-1" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-800 rounded-lg p-6">
            <h3 className="text-xl font-semibold mb-4">Seus Artistas Favoritos</h3>
            <div className="space-y-3">
              {topArtists.map((artist, index) => (
                <div key={artist.id} className="flex items-center space-x-3 p-3 rounded-lg bg-gray-700 hover:bg-gray-600 transition-colors">
                  <span className="text-spotify-green font-bold w-6">{index + 1}</span>
                  {artist.images[0] && (
                    <img
                      src={artist.images[0].url}
                      alt={artist.name}
                      className="w-12 h-12 rounded-full"
                    />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{artist.name}</p>
                    <p className="text-gray-400 text-sm">
                      {artist.genres.slice(0, 2).join(', ')}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-gray-400">Pop. {artist.popularity}/100</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="mt-8">
          <SpotifyDataImport />
        </div>
      </main>
    </div>
  )
}
