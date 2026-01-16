export interface Indicator {
  id: string;
  candle_id: string;
  ema20: string;
  ema50: string;
  ema200: string;
  range_size: string;
  body_size: string;
  upper_wick: string;
  lower_wick: string;
  mid_price: string;
  last_swing_high_price: string | null;
  last_swing_low_price: string | null;
  computed_at: string;
}
