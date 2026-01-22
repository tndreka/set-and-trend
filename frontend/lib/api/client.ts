import axios from 'axios';
import { APIResponse } from '@/types/api';
import { Candle } from '@/types/candle';
import { Indicator } from '@/types/indicator';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
  timeout: 10000,
});

// Auth types
export interface SignupRequest {
  username: string;
  email: string;
  password: string;
  name?: string;
  surname?: string;
}

export interface LoginRequest {
  username_or_email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: {
    id: string;
    username: string;
    email: string;
    name?: string;
    surname?: string;
  };
}

export const apiClient = {
  // Candles & Indicators
  getLatestCandles: (limit = 600) =>
    api.get<APIResponse<Candle[]>>(`/api/candles/latest?limit=${limit}`),

  getLatestIndicators: (limit = 600) =>
    api.get<APIResponse<Indicator[]>>(`/api/indicators/latest?limit=${limit}`),

  // Authentication
  signup: (data: SignupRequest) =>
    api.post<AuthResponse>('/api/auth/signup', data),

  login: (data: LoginRequest) =>
    api.post<AuthResponse>('/api/auth/login', data),
};
