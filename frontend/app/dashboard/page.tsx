'use client';

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import CandleChart from '@/components/charts/CandleChart';
import IndicatorTable from '@/components/tables/IndicatorTable';

export default function Dashboard() {
  const { data: candles, isLoading: candlesLoading, error: candlesError } = useQuery({
    queryKey: ['candles'],
    queryFn: () => apiClient.getLatestCandles(600),
  });

  const { data: indicators, isLoading: indicatorsLoading, error: indicatorsError } = useQuery({
    queryKey: ['indicators'],
    queryFn: () => apiClient.getLatestIndicators(600),
  });

  if (candlesLoading || indicatorsLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-xl">Loading data...</div>
      </div>
    );
  }

  if (candlesError || indicatorsError) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-xl text-red-600">
          Error loading data: {candlesError?.message || indicatorsError?.message}
        </div>
      </div>
    );
  }

  const candleCount = candles?.data?.data?.length || 0;
  const indicatorCount = indicators?.data?.data?.length || 0;

  return (
    <div className="p-8 space-y-8">
      <h1 className="text-4xl font-bold">Set The Trend - Dashboard</h1>
      
      <div className="grid grid-cols-2 gap-4">
        <div className="p-6 border rounded-lg">
          <h2 className="text-2xl font-semibold mb-2">Candles Loaded</h2>
          <p className="text-5xl font-bold text-blue-600">{candleCount}</p>
        </div>
        
        <div className="p-6 border rounded-lg">
          <h2 className="text-2xl font-semibold mb-2">Indicators Computed</h2>
          <p className="text-5xl font-bold text-green-600">{indicatorCount}</p>
        </div>
      </div>
	
	{/* Chart */}
      <CandleChart 
        candles={candles?.data?.data || []} 
        indicators={indicators?.data?.data || []}
      />

	{/* Indicator Table */}
      <IndicatorTable data={indicators?.data?.data || []} />

      <div className="p-6 border rounded-lg">
        <h2 className="text-xl font-semibold mb-4">Backend Status</h2>
        <p className="text-green-600">✓ Connected to API</p>
        <p className="text-sm text-gray-600 mt-2">
          API: {process.env.NEXT_PUBLIC_API_URL}
        </p>
      </div>
    </div>
  );
}
