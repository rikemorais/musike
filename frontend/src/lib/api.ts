import axios from 'axios';
import { SpotifyUser, SpotifyTrack, SpotifyArtist, UserAnalytics, AuthResponse } from '@/types/spotify';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000';

export const api = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('musike_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  const spotifyToken = localStorage.getItem('spotify_token');
  if (spotifyToken) {
    config.headers['Spotify-Token'] = spotifyToken;
  }

  return config;
});

export class AuthService {
  static async getSpotifyAuthUrl(): Promise<{ auth_url: string; state: string }> {
    const response = await api.get('/auth/spotify');
    return response.data;
  }

  static async handleCallback(code: string): Promise<AuthResponse> {
    const response = await api.get(`/auth/callback?code=${code}`);
    
    console.log('Full API response:', response.data);
    
    const { access_token, spotify_token } = response.data;
    console.log('Extracted tokens - JWT:', access_token, 'Spotify:', spotify_token);
    
    if (access_token && spotify_token) {
      localStorage.setItem('musike_token', access_token);
      localStorage.setItem('spotify_token', spotify_token);
      console.log('Tokens saved to localStorage');
    } else {
      console.error('Missing tokens in response:', { access_token, spotify_token });
    }

    return response.data;
  }

  static async refreshToken(refreshToken: string): Promise<{ access_token: string }> {
    const response = await api.post('/auth/refresh', { refresh_token: refreshToken });
    return response.data;
  }

  static logout() {
    localStorage.removeItem('musike_token');
    localStorage.removeItem('spotify_token');
    localStorage.removeItem('user_data');
  }

  static isAuthenticated(): boolean {
    return !!localStorage.getItem('musike_token');
  }
}

export class SpotifyAPIService {
  static async getUserProfile(): Promise<SpotifyUser> {
    const response = await api.get('/user/profile');
    return response.data;
  }

  static async getTopTracks(timeRange: string = 'medium_term', limit: number = 20): Promise<{ items: SpotifyTrack[] }> {
    const response = await api.get(`/user/top-tracks?time_range=${timeRange}&limit=${limit}`);
    return response.data;
  }

  static async getTopArtists(timeRange: string = 'medium_term', limit: number = 20): Promise<{ items: SpotifyArtist[] }> {
    const response = await api.get(`/user/top-artists?time_range=${timeRange}&limit=${limit}`);
    return response.data;
  }

  static async getListeningHistory(limit: number = 50): Promise<any> {
    const response = await api.get(`/user/listening-history?limit=${limit}`);
    return response.data;
  }

  static async getUserAnalytics(timeFilter: string = '6months'): Promise<UserAnalytics> {
    const response = await api.get(`/user/analytics?time_filter=${timeFilter}`);
    return response.data;
  }

  static async getRecommendations(): Promise<any> {
    const response = await api.get('/user/recommendations');
    return response.data;
  }

  static async getRecentlyPlayed(limit: number = 5): Promise<{ items: Array<{ track: SpotifyTrack, played_at: string }> }> {
    const response = await api.get(`/user/recently-played?limit=${limit}`);
    return response.data;
  }
}

export default api;
