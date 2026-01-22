'use client';

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { useEffect, useState } from 'react';
import { TrendingUp, BarChart3, BookOpen, User, Settings } from 'lucide-react';
import LoadingScreen from '@/components/ui/LoadingScreen'; 
import CandleChart from '@/components/charts/CandleChart';
import IndicatorTable from '@/components/tables/IndicatorTable';

export default function Dashboard() {
  const [activeSection, setActiveSection] = useState('markets');
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
      <div className="flex items-center justify-center min-h-screen bg-gray-950">
        <div className="text-xl text-red-400">
          Error loading data: {candlesError?.message || indicatorsError?.message}
        </div>
      </div>
    );
  }

  const candleCount = candles?.data?.data?.length || 0;
  const indicatorCount = indicators?.data?.data?.length || 0;

  const navItems = [
    { id: 'markets', label: 'Markets', icon: BarChart3 },
    { id: 'journal', label: 'Journal', icon: BookOpen },
    { id: 'profile', label: 'Profile', icon: User },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];

  return (
    <div className="min-h-screen bg-gray-950 text-white flex flex-col md:flex-row">
      {/* Background Effects */}
      <div className="fixed inset-0 bg-[linear-gradient(to_right,#ffffff08_1px,transparent_1px),linear-gradient(to_bottom,#ffffff08_1px,transparent_1px)] bg-[size:64px_64px] pointer-events-none" />
      <div className="fixed top-1/4 right-1/4 w-[500px] h-[500px] bg-green-500/10 rounded-full blur-[120px] pointer-events-none" />

      {/* Mobile Bottom Navbar / Desktop Vertical Navbar */}
      <aside className="fixed bottom-0 left-0 right-0 h-16 md:h-screen md:w-20 md:top-0 md:bottom-auto md:right-auto border-t md:border-t-0 md:border-r border-white/10 bg-gray-950/95 backdrop-blur-xl z-50 flex md:flex-col items-center justify-around md:justify-start md:py-8">
        {/* Logo - hidden on mobile */}
        <div className="hidden md:block mb-12 relative group cursor-pointer">
          <div className="absolute inset-0 bg-green-500 blur-lg opacity-50 group-hover:opacity-70 transition-opacity" />
          <TrendingUp className="w-8 h-8 text-green-400 relative" />
        </div>

        {/* Navigation Items */}
        <nav className="flex md:flex-col items-center justify-around md:justify-start w-full md:w-auto md:space-y-6 md:flex-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeSection === item.id;
            return (
              <button
                key={item.id}
                onClick={() => setActiveSection(item.id)}
                className={`relative group flex flex-col items-center space-y-1 transition-all ${
                  isActive ? 'text-green-400' : 'text-gray-500 hover:text-white'
                }`}
                title={item.label}
              >
                {isActive && (
                  <div className="absolute -left-10 top-1/2 -translate-y-1/2 w-1 h-8 bg-green-400 rounded-r" />
                )}
                <Icon className="w-6 h-6" />
                <span className="text-[10px] font-medium">{item.label}</span>
              </button>
            );
          })}
        </nav>
      </aside>

      {/* Main Content */}
      <main className="flex-1 md:ml-20 mb-16 md:mb-0 relative">
        <div className="p-4 md:p-8 space-y-4 md:space-y-8">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl md:text-4xl font-bold tracking-tight">
                {navItems.find(item => item.id === activeSection)?.label}
              </h1>
              <p className="text-gray-400 mt-1 md:mt-2 text-sm md:text-base">
                {activeSection === 'markets' && 'Weekly trend analysis and trading signals'}
                {activeSection === 'journal' && 'Track your trades and performance'}
                {activeSection === 'profile' && 'Manage your account information'}
                {activeSection === 'settings' && 'Configure your preferences'}
              </p>
            </div>
          </div>

          {/* Content Sections */}
          {activeSection === 'markets' && (
            <>
              {/* Stats Cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 md:gap-4">
                <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-4 md:p-6">
                  <h2 className="text-sm md:text-lg font-semibold mb-1 md:mb-2 text-gray-400">Candles Loaded</h2>
                  <p className="text-3xl md:text-5xl font-bold text-green-400">{candleCount}</p>
                </div>
                
                <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-4 md:p-6">
                  <h2 className="text-sm md:text-lg font-semibold mb-1 md:mb-2 text-gray-400">Indicators Computed</h2>
                  <p className="text-3xl md:text-5xl font-bold text-green-400">{indicatorCount}</p>
                </div>
              </div>

              {/* Chart */}
              <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-3 md:p-6 overflow-x-auto">
                <CandleChart 
                  candles={candles?.data?.data || []} 
                  indicators={indicators?.data?.data || []}
                />
              </div>

              {/* Indicator Table */}
              <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-3 md:p-6 overflow-x-auto">
                <IndicatorTable 
                  indicators={indicators?.data?.data || []}
                  candles={candles?.data?.data || []}
                />
              </div>
            </>
          )}

          {activeSection === 'journal' && (
            <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-8 md:p-12 text-center">
              <BookOpen className="w-12 h-12 md:w-16 md:h-16 text-green-400 mx-auto mb-4" />
              <h3 className="text-xl md:text-2xl font-bold mb-2">Trade Journal</h3>
              <p className="text-gray-400 text-sm md:text-base">Coming soon - Document your trades systematically</p>
            </div>
          )}

          {activeSection === 'profile' && (
            <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-8 md:p-12 text-center">
              <User className="w-12 h-12 md:w-16 md:h-16 text-green-400 mx-auto mb-4" />
              <h3 className="text-xl md:text-2xl font-bold mb-2">Profile</h3>
              <p className="text-gray-400 text-sm md:text-base">Manage your account information</p>
            </div>
          )}

          {activeSection === 'settings' && (
            <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-xl md:rounded-2xl p-8 md:p-12 text-center">
              <Settings className="w-12 h-12 md:w-16 md:h-16 text-green-400 mx-auto mb-4" />
              <h3 className="text-xl md:text-2xl font-bold mb-2">Settings</h3>
              <p className="text-gray-400 text-sm md:text-base">Configure your preferences</p>
            </div>
          )}
        </div>
      </main>
    </div>
  );
 }

