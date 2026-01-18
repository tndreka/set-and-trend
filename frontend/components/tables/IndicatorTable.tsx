'use client';

import { useMemo } from 'react';
import { format, parseISO } from 'date-fns';
import { Indicator } from '@/types/indicator';
import { Candle } from '@/types/candle';

interface IndicatorTableProps {
  indicators: Indicator[];
  candles: Candle[];
}

export default function IndicatorTable({ indicators, candles }: IndicatorTableProps) {
  const tableData = useMemo(() => {
    // Create map of candle_id -> candle
    const candleMap = new Map(candles.map(c => [c.id, c]));

    return indicators
      .map(ind => {
        const candle = candleMap.get(ind.candle_id);
        if (!candle) return null;

        return {
          id: ind.id,
          timestamp: candle.timestamp_utc,
          date: format(parseISO(candle.timestamp_utc), 'MMM dd, yyyy'),
          ema20: parseFloat(ind.ema20),
          ema50: parseFloat(ind.ema50),
          ema200: parseFloat(ind.ema200),
        };
      })
      .filter(Boolean)
      .sort((a, b) => new Date(b!.timestamp).getTime() - new Date(a!.timestamp).getTime())
      .slice(0, 260); // Last 5 years (52 weeks × 5)
  }, [indicators, candles]);

  const getTrendLabel = (ema50: number, ema200: number) => {
    if (ema50 > ema200) {
      return <span className="px-2 py-1 bg-green-900 text-green-300 rounded text-sm font-semibold">BULLISH</span>;
    }
    return <span className="px-2 py-1 bg-red-900 text-red-300 rounded text-sm font-semibold">BEARISH</span>;
  };

  return (
    <div className="p-6 border rounded-lg">
      <h2 className="text-2xl font-bold mb-4">Weekly Indicators History</h2>
      <p className="text-sm text-gray-600 mb-4">
        Showing last 5 years ({tableData.length} weeks) • Trend = EMA50 vs EMA200 • Scroll to see more
      </p>

      {/* Scrollable container with max height */}
      <div className="overflow-auto max-h-[600px] border border-gray-700 rounded">
        <table className="w-full text-left">
          <thead className="border-b border-gray-700 bg-gray-900 sticky top-0">
            <tr>
              <th className="py-3 px-4 font-semibold">Week</th>
              <th className="py-3 px-4 text-right font-semibold">EMA 20</th>
              <th className="py-3 px-4 text-right font-semibold">EMA 50</th>
              <th className="py-3 px-4 text-right font-semibold">EMA 200</th>
              <th className="py-3 px-4 font-semibold">Trend</th>
            </tr>
          </thead>
          <tbody>
            {tableData.map((row, idx) => (
              <tr 
                key={row!.id} 
                className={`border-b border-gray-800 hover:bg-gray-800 transition-colors ${
                  idx % 2 === 0 ? 'bg-gray-900/50' : ''
                }`}
              >
                <td className="py-3 px-4 text-sm">{row!.date}</td>
                <td className="py-3 px-4 text-right font-mono text-sm">{row!.ema20.toFixed(5)}</td>
                <td className="py-3 px-4 text-right font-mono text-sm">{row!.ema50.toFixed(5)}</td>
                <td className="py-3 px-4 text-right font-mono text-sm">{row!.ema200.toFixed(5)}</td>
                <td className="py-3 px-4">{getTrendLabel(row!.ema50, row!.ema200)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-gray-500 mt-2">
         Tip: Scroll down to see historical data back to {tableData[tableData.length - 1]?.date}
      </p>
    </div>
  );
}
