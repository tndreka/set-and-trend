'use client';

import { useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { format, parseISO } from 'date-fns';
import { Candle } from '@/types/candle';
import { Indicator } from '@/types/indicator';

interface CandleChartProps {
  candles: Candle[];
  indicators: Indicator[];
}

export default function CandleChart({ candles, indicators }: CandleChartProps) {
  const chartData = useMemo(() => {
    // Create a map of candle_id -> indicator
    const indicatorMap = new Map(
      indicators.map(ind => [ind.candle_id, ind])
    );

    // Merge candles with their indicators
    const merged = candles
      .map(candle => {
        const indicator = indicatorMap.get(candle.id);
        if (!indicator) return null;

        return {
          timestamp: candle.timestamp_utc,
          date: format(parseISO(candle.timestamp_utc), 'MMM dd, yyyy'),
          close: parseFloat(candle.close),
          ema20: parseFloat(indicator.ema20),
          ema50: parseFloat(indicator.ema50),
          ema200: parseFloat(indicator.ema200),
        };
      })
      .filter(Boolean)
      .sort((a, b) => new Date(a!.timestamp).getTime() - new Date(b!.timestamp).getTime());

    return merged;
  }, [candles, indicators]);

  if (chartData.length === 0) {
    return (
      <div className="p-8 border rounded-lg">
        <p className="text-red-600">No data available for chart</p>
      </div>
    );
  }

  return (
    <div className="p-6 border rounded-lg">
      <h2 className="text-2xl font-bold mb-4">EURUSD Weekly Chart (2015-2025)</h2>
      <p className="text-sm text-gray-600 mb-4">
        {chartData.length} candles • Close price with EMA 20/50/200
      </p>
      
      <ResponsiveContainer width="100%" height={500}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
          <XAxis 
            dataKey="date" 
            tick={{ fontSize: 12 }}
            interval="preserveStartEnd"
          />
          <YAxis 
            domain={['auto', 'auto']}
            tick={{ fontSize: 12 }}
            tickFormatter={(value) => value.toFixed(5)}
          />
          <Tooltip 
            contentStyle={{ backgroundColor: '#1f2937', border: '1px solid #374151' }}
            labelStyle={{ color: '#fff' }}
          />
          <Legend />
          
          {/* Close Price */}
          <Line 
            type="monotone" 
            dataKey="close" 
            stroke="#3b82f6" 
            strokeWidth={2}
            dot={false}
            name="Close"
          />
          
          {/* EMA 20 */}
          <Line 
            type="monotone" 
            dataKey="ema20" 
            stroke="#10b981" 
            strokeWidth={1.5}
            dot={false}
            name="EMA 20"
          />
          
          {/* EMA 50 */}
          <Line 
            type="monotone" 
            dataKey="ema50" 
            stroke="#f59e0b" 
            strokeWidth={1.5}
            dot={false}
            name="EMA 50"
          />
          
          {/* EMA 200 */}
          <Line 
            type="monotone" 
            dataKey="ema200" 
            stroke="#ef4444" 
            strokeWidth={1.5}
            dot={false}
            name="EMA 200"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
