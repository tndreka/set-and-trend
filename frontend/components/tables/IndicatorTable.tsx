'use client';

import { useMemo } from 'react';
import { format, parseISO } from 'date-fns';
import { Indicator } from '@/types/indicator';

interface IndicatorTableProps {
  data: Indicator[];
}

export default function IndicatorTable({ data }: IndicatorTableProps) {
  const tableData = useMemo(() => {
    return data
      .map(ind => ({
        id: ind.id,
        timestamp: ind.computed_at,
        date: format(parseISO(ind.computed_at), 'MMM dd, yyyy'),
        ema20: parseFloat(ind.ema20),
        ema50: parseFloat(ind.ema50),
        ema200: parseFloat(ind.ema200),
      }))
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, 10); // Latest 10
  }, [data]);

  const getTrendLabel = (ema50: number, ema200: number) => {
    if (ema50 > ema200) {
      return <span className="px-2 py-1 bg-green-900 text-green-300 rounded text-sm font-semibold">BULLISH</span>;
    }
    return <span className="px-2 py-1 bg-red-900 text-red-300 rounded text-sm font-semibold">BEARISH</span>;
  };

  return (
    <div className="p-6 border rounded-lg">
      <h2 className="text-2xl font-bold mb-4">Latest Weekly Indicators</h2>
      <p className="text-sm text-gray-600 mb-4">
        Showing last 10 weeks • Trend = EMA50 vs EMA200
      </p>

      <div className="overflow-x-auto">
        <table className="w-full text-left">
          <thead className="border-b border-gray-700">
            <tr>
              <th className="pb-3 pr-4">Week</th>
              <th className="pb-3 pr-4 text-right">EMA 20</th>
              <th className="pb-3 pr-4 text-right">EMA 50</th>
              <th className="pb-3 pr-4 text-right">EMA 200</th>
              <th className="pb-3">Trend</th>
            </tr>
          </thead>
          <tbody>
            {tableData.map((row, idx) => (
              <tr key={row.id} className={idx % 2 === 0 ? 'bg-gray-900' : ''}>
                <td className="py-3 pr-4">{row.date}</td>
                <td className="py-3 pr-4 text-right font-mono text-sm">{row.ema20.toFixed(5)}</td>
                <td className="py-3 pr-4 text-right font-mono text-sm">{row.ema50.toFixed(5)}</td>
                <td className="py-3 pr-4 text-right font-mono text-sm">{row.ema200.toFixed(5)}</td>
                <td className="py-3">{getTrendLabel(row.ema50, row.ema200)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
