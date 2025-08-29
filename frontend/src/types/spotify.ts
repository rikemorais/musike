export interface SpotifyUser {
  id: string;
  display_name: string;
  email: string;
  country: string;
  followers: {
    total: number;
  };
  images: Array<{
    url: string;
  }>;
}

export interface SpotifyTrack {
  id: string;
  name: string;
  artists: SpotifyArtist[];
  album: SpotifyAlbum;
  duration_ms: number;
  popularity: number;
  preview_url: string;
  external_ids: {
    isrc: string;
  };
}

export interface SpotifyArtist {
  id: string;
  name: string;
  genres: string[];
  popularity: number;
  images: Array<{
    url: string;
  }>;
}

export interface SpotifyAlbum {
  id: string;
  name: string;
  release_date: string;
  images: Array<{
    url: string;
  }>;
}

export interface UserAnalytics {
  user_id: string;
  total_listening_time_ms: number;
  actual_listening_time_ms: number;
  total_plays: number;
  average_play_time_ms: number;
  average_track_popularity: number;
  avg_listening_percentage: number;
  top_genres: GenreStats[];
  listening_patterns: ListeningPatterns;
  diversity_score: number;
  recent_activity: ActivityPoint[];
  monthly_stats: Record<string, MonthStats>;
}

export interface GenreStats {
  genre: string;
  percentage: number;
  track_count: number;
  play_count: number;
  total_time_ms: number;
}

export interface ListeningPatterns {
  peak_hours: number[];
  weekday_usage: number[];
  seasonality: Record<string, number>;
}

export interface ActivityPoint {
  date: string;
  track_count: number;
  unique_tracks: number;
  duration_ms: number;
}

export interface MonthStats {
  tracks_played: number;
  unique_artists: number;
  top_genre: string;
  avg_daily_minutes: number;
}

export interface AuthResponse {
  access_token: string;
  user: SpotifyUser;
  spotify_token: string;
  expires_in: string;
}
