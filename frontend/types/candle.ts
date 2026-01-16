export interface Candle {
  id: string;
  timestamp_utc: string;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: number | null;
  created_at: string;
}
