'use client';

import { useEffect, useState } from 'react';

export default function LoadingScreen() {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    let currentProgress = 0;
    
    const updateProgress = () => {
      if (currentProgress < 20) {
        currentProgress += 2;
      } else if (currentProgress < 50) {
        currentProgress += 2.5;
      } else if (currentProgress < 80) {
        currentProgress += 2;
      } else if (currentProgress < 95) {
        currentProgress += 1;
      } else if (currentProgress < 99) {
        currentProgress += 0.5;
      } else {
        currentProgress = 100;
      }
      
      setProgress(Math.min(currentProgress, 100));
    };
    
    const interval = setInterval(updateProgress, 80);
    
    return () => clearInterval(interval);
  }, []);

  // 13 candles in uptrend (bullish pattern)
   const candles = [
    { low: 5, high: 18, open: 7, close: 16 },   // Green
    { low: 14, high: 26, open: 15, close: 24 }, // Green
    { low: 22, high: 32, open: 23, close: 20 }, // Red (pullback)
    { low: 28, high: 40, open: 30, close: 38 }, // Green
    { low: 36, high: 48, open: 37, close: 46 }, // Green
    { low: 44, high: 54, open: 45, close: 42 }, // Red (pullback)
    { low: 50, high: 62, open: 52, close: 60 }, // Green
    { low: 58, high: 70, open: 59, close: 68 }, // Green
    { low: 66, high: 76, open: 67, close: 64 }, // Red (pullback)
    { low: 72, high: 84, open: 74, close: 82 }, // Green
    { low: 80, high: 90, open: 81, close: 88 }, // Green
    { low: 86, high: 95, open: 87, close: 93 }, // Green
    { low: 91, high: 98, open: 92, close: 97 }, // Green (final push higher)
  ];

  return (
    <div className="fixed inset-0 bg-black flex items-center justify-center overflow-hidden">
      {/* Background Grid */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff08_1px,transparent_1px),linear-gradient(to_bottom,#ffffff08_1px,transparent_1px)] bg-[size:64px_64px]" />
      
      {/* Green Glow */}
      <div className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-green-500/20 rounded-full blur-[120px]" />
      
      <div className="text-center space-y-16 px-8 relative z-10">
        
        {/* Title */}
        <h1 className="text-6xl font-bold tracking-tight mb-8">
          <span className="text-white">SET</span>
          <span className="text-green-400">&</span>
          <span className="text-white">TREND</span>
        </h1>

        {/* 8 Candles in Uptrend */}
        <div className="flex items-end justify-center space-x-6 h-80">
          {candles.map((candle, idx) => {
            // Each candle fills based on progress (0-100 split into 8 segments)
            const candleStartPercent = (idx / 13) * 100;
            const candleEndPercent = ((idx + 1) / 13) * 100;
            
            // Calculate fill percentage for this specific candle
            const fillPercent = Math.max(0, Math.min(100, 
              ((progress - candleStartPercent) / (candleEndPercent - candleStartPercent)) * 100
            ));
            
            const isGreen = candle.close > candle.open;
            const bodyHeight = Math.abs(candle.close - candle.open);
            const bodyBottom = Math.min(candle.open, candle.close);
            
            return (
  <div
    key={idx}
    className="relative flex flex-col items-center"
    style={{
      height: '100%',
      animation: `slideUp 0.4s ease-out ${idx * 0.1}s both`,
    }}
  >
    <div className="flex-1" /> {/* Spacer to push candle to bottom */}
    
    {/* Top Wick */}
    <div 
      className="w-1 bg-gray-600"
      style={{ 
        height: `${candle.high - Math.max(candle.open, candle.close)}%`,
      }}
    />
    
    {/* Candle Body */}
    <div
      className={`w-14 relative border-2 transition-all duration-300 ${
        isGreen 
          ? 'border-green-500/50' 
          : 'border-red-500/50'
      }`}
      style={{ 
        height: `${bodyHeight}%`,
      }}
    >
      {/* Washed Background (always visible) */}
      <div
        className={`absolute inset-0 ${
          isGreen 
            ? 'bg-green-500/10' 
            : 'bg-red-500/10'
        }`}
      />
      
      {/* Progressive Fill from Bottom to Top */}
      <div
        className={`absolute bottom-0 left-0 right-0 transition-all duration-500 ease-out ${
          isGreen 
            ? 'bg-gradient-to-t from-green-500 to-green-400' 
            : 'bg-gradient-to-t from-red-500 to-red-400'
        }`}
        style={{
          height: `${fillPercent}%`,
          opacity: 0.2 + (fillPercent / 100) * 0.8,
          boxShadow: fillPercent > 95 
            ? `0 0 20px ${isGreen ? 'rgba(34, 197, 94, 0.6)' : 'rgba(239, 68, 68, 0.6)'}`
            : 'none',
        }}
      />

      {/* Pop Glow at 100% */}
      {fillPercent === 100 && (
        <div
          className={`absolute inset-0 ${
            isGreen ? 'bg-green-400' : 'bg-red-400'
          }`}
          style={{
            animation: 'popGlow 0.5s ease-out',
            opacity: 0,
          }}
        />
      )}
    </div>
    
    {/* Bottom Wick - NOW CONNECTED */}
    <div 
      className="w-1 bg-gray-600"
      style={{ 
        height: `${Math.min(candle.open, candle.close) - candle.low}%`,
      }}
    />
    
             <div style={{ height: `${bodyBottom}%` }} /> {/* Bottom spacing */}
           </div>
	   );
          })}
        </div>

        {/* Horizontal Progress Bar */}
        <div className="w-full max-w-3xl mx-auto space-y-3">
          <div className="h-3 bg-gray-800 rounded-full overflow-hidden shadow-inner">
            <div
              className="h-full bg-gradient-to-r from-green-500 via-emerald-500 to-green-600 transition-all duration-300 ease-out"
              style={{ 
                width: `${progress}%`,
                boxShadow: '0 0 15px rgba(34, 197, 94, 0.5)',
              }}
            />
          </div>
          <p className="text-gray-400 text-xl font-medium">
            {Math.round(progress)}%
          </p>
        </div>

        <p className="text-gray-500 text-lg">Loading your data...</p>

      </div>

      {/* CSS Animations */}
      <style jsx>{`
        @keyframes slideUp {
          from {
            opacity: 0;
            transform: translateY(30px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes popGlow {
          0% {
            opacity: 0.8;
            transform: scale(1);
          }
          50% {
            opacity: 0.4;
            transform: scale(1.08);
          }
          100% {
            opacity: 0;
            transform: scale(1.15);
          }
        }
      `}</style>
    </div>
  );
}
