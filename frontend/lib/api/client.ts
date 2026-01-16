import axios from 'axios';
import { APIResponse } from '@/types/api';
import { Candle } from '@/types/candle';
import { Indicator } from '@/types/indicator';

const API_BASE = 'http://164.92.229.200:8080';

export const api = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
  timeout: 10000,
});

export const apiClient = {
  getLatestCandles: (limit = 600) =>
    api.get<APIResponse<Candle[]>>(`/api/candles/latest?limit=${limit}`),

  getLatestIndicators: (limit = 600) =>
    api.get<APIResponse<Indicator[]>>(`/api/indicators/latest?limit=${limit}`),
};
