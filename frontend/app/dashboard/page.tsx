'use client';

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { useEffect, useState } from 'react';
import LoadingScreen from '@/components/ui/LoadingScreen'; 
import CandleChart from '@/components/charts/CandleChart';
import IndicatorTable from '@/components/tables/IndicatorTable';

export default function Dashboard() {
  const [showLoading, setShowLoading] = useState(true);
  const [minLoadingComplete, setMinLoadingComplete] = useState(false);
  
  const { data: candles, isLoading: candlesLoading, error: candlesError } = useQuery({
    queryKey: ['candles'],
    queryFn: () => apiClient.getLatestCandles(600),
  });

  // loading time: 12 seconds
  useEffect(() => {
    const timer = setTimeout(() => {
      setMinLoadingComplete(true);
    }, 12000);

    return () => clearTimeout(timer);
  }, []);

  const { data: indicators, isLoading: indicatorsLoading, error: indicatorsError } = useQuery({
    queryKey: ['indicators'],
    queryFn: () => apiClient.getLatestIndicators(600),
  });

   useEffect(() => {
    if (!candlesLoading && !indicatorsLoading && minLoadingComplete) {
      setShowLoading(false);
    }
  }, [candlesLoading, indicatorsLoading, minLoadingComplete]); 

  if (showLoading || candlesLoading || indicatorsLoading) {
    return <LoadingScreen />;
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

	   {/* Indicator Table - NOW RECEIVES BOTH */}
      <IndicatorTable 
        indicators={indicators?.data?.data || []}
        candles={candles?.data?.data || []}
      />


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

