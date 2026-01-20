'use client';

import Link from 'next/link';
import { ArrowRight, TrendingUp, TrendingDown } from 'lucide-react';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-gray-950 text-white overflow-hidden">
      {/* Fixed Navbar */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-gray-950/80 backdrop-blur-xl border-b border-white/10">
        <div className="max-w-[1400px] mx-auto px-8 py-5 flex items-center justify-between">
          {/* Logo */}
          <Link href="/" className="flex items-center space-x-3 group">
            <div className="relative">
              <div className="absolute inset-0 bg-green-500 blur-lg opacity-50 group-hover:opacity-70 transition-opacity" />
              <TrendingUp className="w-7 h-7 text-green-400 relative" />
            </div>
            <span className="text-xl font-bold tracking-tight">
              SET<span className="text-green-400">&</span>TREND
            </span>
          </Link>

          {/* Right CTAs */}
          <div className="flex items-center space-x-3">
            <Link
              href="/signup"
              className="px-5 py-2 text-sm font-medium text-gray-400 hover:text-white transition-colors"
            >
              Sign Up
            </Link>
            <Link
              href="/login"
              className="px-5 py-2 bg-white text-black rounded-lg font-medium text-sm hover:bg-gray-200 transition-all"
            >
              Log In
            </Link>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="pt-20">
        {/* Hero Section */}
        {/* Hero Section */}
        <section className="min-h-screen flex items-center relative">
          {/* Background Grid */}
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff08_1px,transparent_1px),linear-gradient(to_bottom,#ffffff08_1px,transparent_1px)] bg-[size:64px_64px]" />
          
          {/* Green Glow */}
          <div className="absolute top-1/4 left-1/4 w-[500px] h-[500px] bg-green-500/20 rounded-full blur-[120px]" />
          
          <div className="max-w-[1400px] mx-auto px-8 py-20 relative z-10">
            <div className="grid grid-cols-12 gap-8 items-center">
              {/* Left: Text - Takes 7 columns */}
              <div className="col-span-12 lg:col-span-7 space-y-8">
                {/* Badge */}
                <div className="inline-flex items-center space-x-2 px-4 py-2 bg-green-500/10 border border-green-500/20 rounded-full">
                  <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
                  <span className="text-sm font-medium text-green-400">Rule-Based Trading System</span>
                </div>

                <h1 className="text-7xl lg:text-[80px] leading-[0.95] font-bold tracking-tight">
                  Master forex<br />
                  with <span className="text-green-400 italic font-light">discipline</span>
                </h1>
                
                <p className="text-xl text-gray-400 leading-relaxed max-w-[500px]">
                  Weekly charts. EMA signals. Clear PASS/FAIL rules. No guesswork, just data-driven decisions.
                </p>

                <div className="flex items-center space-x-4">
                  <Link
                    href="/login"
                    className="group px-8 py-4 bg-green-500 text-black rounded-xl font-semibold flex items-center space-x-2 hover:bg-green-400 transition-all"
                  >
                    <span>Start Trading</span>
                    <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                  </Link>
                  <Link
                    href="/dashboard"
                    className="px-8 py-4 border border-white/10 rounded-xl font-semibold hover:bg-white/5 transition-all"
                  >
                    View Demo
                  </Link>
                </div>

                {/* Mini Stats */}
                <div className="flex items-center space-x-8 pt-8 border-t border-white/10">
                  <div>
                    <p className="text-3xl font-bold">554</p>
                    <p className="text-sm text-gray-500">Weeks analyzed</p>
                  </div>
                  <div>
                    <p className="text-3xl font-bold">28</p>
                    <p className="text-sm text-gray-500">Forex pairs</p>
                  </div>
                  <div>
                    <p className="text-3xl font-bold">100%</p>
                    <p className="text-sm text-gray-500">Rule-driven</p>
                  </div>
                </div>
              </div>

              {/* Right: Dashboard Screenshot - Takes 5 columns */}
              <div className="col-span-12 lg:col-span-5">
                <div className="relative">
                  {/* Glowing border effect */}
                  <div className="absolute -inset-0.5 bg-gradient-to-r from-green-500 to-emerald-500 rounded-2xl blur opacity-30" />
                  
                  <div className="relative bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-6 shadow-2xl">
                    {/* Real Chart Screenshot */}
                    <div className="aspect-[4/3] bg-black/50 rounded-xl mb-4 border border-white/5 overflow-hidden">
                      <img 
                        src="/weekly_chart.png" 
                        alt="EURUSD Weekly Chart" 
                        className="w-full h-full object-cover"
                      />
                    </div>

                    {/* Stats Grid */}
                    <div className="grid grid-cols-3 gap-3">
                      <div className="bg-black/50 rounded-lg p-3 border border-green-500/20">
                        <p className="text-xs text-gray-500 mb-1">EMA 50</p>
                        <p className="text-green-400 font-bold">↑ BULL</p>
                      </div>
                      <div className="bg-black/50 rounded-lg p-3 border border-white/10">
                        <p className="text-xs text-gray-500 mb-1">Range</p>
                        <p className="font-bold">154 pips</p>
                      </div>
                      <div className="bg-black/50 rounded-lg p-3 border border-green-500/20">
                        <p className="text-xs text-gray-500 mb-1">Signal</p>
                        <p className="text-green-400 font-bold">PASS</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Markets Preview Section */}
        {/* Markets Preview Section */}
        <section className="relative py-32">
          {/* Diagonal Background */}
          <div className="absolute inset-0 bg-gradient-to-br from-green-950/20 to-transparent transform -skew-y-2 origin-top-left" />
          
          <div className="max-w-[1400px] mx-auto px-8 relative z-10">
            {/* Section Header */}
            <div className="mb-16">
              <h2 className="text-6xl font-bold mb-4">Live markets</h2>
              <p className="text-xl text-gray-400">Real-time trend analysis across major pairs</p>
            </div>

            {/* Market Cards - Bento Grid Style */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {/* EURUSD - Large Card (spans 2 columns) */}
              <div className="col-span-2 md:col-span-2 bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/50 transition-all cursor-pointer group">
                <div className="flex items-start justify-between mb-6">
                  <div>
                    <h3 className="text-4xl font-bold mb-2">EURUSD</h3>
                    <p className="text-green-400 text-2xl font-bold">+0.54%</p>
                  </div>
                  <div className="w-12 h-12 bg-green-500/10 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform">
                    <TrendingUp className="w-6 h-6 text-green-400" />
                  </div>
                </div>
                <div className="flex items-center space-x-6 mb-6">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-400 rounded-full" />
                    <span className="text-green-400 font-semibold">BULL</span>
                  </div>
                  <span className="text-gray-500">|</span>
                  <p className="text-sm text-gray-400">W1 EMA ↑</p>
                </div>
                <Link href="/dashboard" className="text-sm font-medium text-gray-400 group-hover:text-green-400 transition-colors">
                  View Chart →
                </Link>
              </div>

              {/* GBPUSD */}
              <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-6 hover:border-red-500/50 transition-all cursor-pointer group">
                <h3 className="text-2xl font-bold mb-2">GBPUSD</h3>
                <p className="text-red-400 text-xl font-bold mb-4">-1.23%</p>
                <div className="flex items-center space-x-2 mb-4">
                  <TrendingDown className="w-5 h-5 text-red-400" />
                  <span className="text-red-400 font-semibold text-sm">BEAR</span>
                </div>
                <p className="text-xs text-gray-500">W1 EMA ↓</p>
              </div>

              {/* USDJPY */}
              <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-6 hover:border-green-500/50 transition-all cursor-pointer group">
                <h3 className="text-2xl font-bold mb-2">USDJPY</h3>
                <p className="text-green-400 text-xl font-bold mb-4">+2.14%</p>
                <div className="flex items-center space-x-2 mb-4">
                  <TrendingUp className="w-5 h-5 text-green-400" />
                  <span className="text-green-400 font-semibold text-sm">BULL</span>
                </div>
                <p className="text-xs text-gray-500">W1 EMA ↑</p>
              </div>

              {/* AUDUSD */}
              <div className="col-span-2 md:col-span-1 bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-6 hover:border-red-500/50 transition-all cursor-pointer group">
                <h3 className="text-2xl font-bold mb-2">AUDUSD</h3>
                <p className="text-red-400 text-xl font-bold mb-4">-0.87%</p>
                <div className="flex items-center space-x-2 mb-4">
                  <TrendingDown className="w-5 h-5 text-red-400" />
                  <span className="text-red-400 font-semibold text-sm">BEAR</span>
                </div>
                <p className="text-xs text-gray-500">W1 EMA ↓</p>
              </div>

              {/* More Markets CTA */}
              <div className="col-span-2 md:col-span-1 bg-gradient-to-br from-green-950/20 to-black border border-green-500/20 rounded-2xl p-6 flex flex-col items-center justify-center text-center cursor-pointer group hover:border-green-500/50 transition-all">
                <TrendingUp className="w-10 h-10 text-green-400 mb-3 group-hover:scale-110 transition-transform" />
                <p className="font-bold mb-1">+24 More</p>
                <p className="text-xs text-gray-500">View all pairs →</p>
              </div>
            </div>
          </div>
        </section>

        {/* Core Features Section */}
        <section className="py-32 relative">
          <div className="absolute right-0 top-1/2 w-[400px] h-[400px] bg-green-500/10 rounded-full blur-[100px]" />
          
          <div className="max-w-[1400px] mx-auto px-8 relative z-10">
            <div className="grid grid-cols-12 gap-16 items-center">
              <div className="col-span-12 lg:col-span-5">
                <h2 className="text-6xl font-bold mb-6">Everything you need</h2>
                <p className="text-xl text-gray-400 leading-relaxed">
                  A complete trading toolkit designed for systematic, rule-based decision making. No emotions, just data.
                </p>
              </div>

              <div className="col-span-12 lg:col-span-7">
                <div className="grid grid-cols-2 gap-6">
                  {/* Markets */}
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/30 transition-all group">
                    <TrendingUp className="w-12 h-12 text-green-400 mb-6 group-hover:scale-110 transition-transform" />
                    <h3 className="text-2xl font-bold mb-3">Markets</h3>
                    <p className="text-gray-400 text-sm leading-relaxed">
                      Track all major forex pairs with real-time weekly trend analysis
                    </p>
                  </div>

                  {/* Charts & Rules */}
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/30 transition-all group">
                    <div className="w-12 h-12 text-green-400 mb-6 group-hover:scale-110 transition-transform flex items-center">
                      <svg className="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                      </svg>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">Charts & Rules</h3>
                    <p className="text-gray-400 text-sm leading-relaxed">
                      Clear PASS/FAIL rules based on EMA signals and price action
                    </p>
                  </div>

                  {/* News/Calendar */}
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/30 transition-all group">
                    <div className="w-12 h-12 text-green-400 mb-6 group-hover:scale-110 transition-transform flex items-center">
                      <svg className="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">News/Calendar</h3>
                    <p className="text-gray-400 text-sm leading-relaxed">
                      Stay ahead with economic events and market-moving updates
                    </p>
                  </div>

                  {/* Trade Journal */}
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/30 transition-all group">
                    <div className="w-12 h-12 text-green-400 mb-6 group-hover:scale-110 transition-transform flex items-center">
                      <svg className="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">Trade Journal</h3>
                    <p className="text-gray-400 text-sm leading-relaxed">
                      Document trades and track performance systematically
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* How It Works Section */}
        <section className="py-32 bg-gradient-to-b from-gray-950/50 to-black relative overflow-hidden">
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff03_1px,transparent_1px),linear-gradient(to_bottom,#ffffff03_1px,transparent_1px)] bg-[size:80px_80px]" />
          
          <div className="max-w-[1400px] mx-auto px-8 relative z-10">
            <div className="text-center mb-20">
              <h2 className="text-6xl font-bold mb-4">How it works</h2>
              <p className="text-xl text-gray-400">Three steps to systematic trading</p>
            </div>

            <div className="relative">
              {/* Connection Line */}
              <div className="absolute top-1/2 left-0 right-0 h-[2px] bg-gradient-to-r from-green-500/20 via-green-500/50 to-green-500/20 hidden md:block" />
              
              <div className="grid grid-cols-1 md:grid-cols-3 gap-8 relative">
                {/* Step 1 */}
                <div className="relative">
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/50 transition-all">
                    <div className="relative inline-block mb-6">
                      <div className="absolute inset-0 bg-green-500 blur-xl opacity-50" />
                      <div className="relative w-16 h-16 bg-green-500 rounded-2xl flex items-center justify-center">
                        <span className="text-3xl font-bold text-black">1</span>
                      </div>
                    </div>
                    <div className="mb-4">
                      <span className="text-7xl font-bold text-gray-800">01</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">Pick Market</h3>
                    <p className="text-gray-400 leading-relaxed">
                      Choose from major forex pairs based on weekly trend signals
                    </p>
                  </div>
                </div>

                {/* Step 2 */}
                <div className="relative">
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/50 transition-all">
                    <div className="relative inline-block mb-6">
                      <div className="absolute inset-0 bg-green-500 blur-xl opacity-50" />
                      <div className="relative w-16 h-16 bg-green-500 rounded-2xl flex items-center justify-center">
                        <span className="text-3xl font-bold text-black">2</span>
                      </div>
                    </div>
                    <div className="mb-4">
                      <span className="text-7xl font-bold text-gray-800">02</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">View Chart</h3>
                    <p className="text-gray-400 leading-relaxed">
                      Analyze weekly charts with EMA indicators and PASS/FAIL rules
                    </p>
                  </div>
                </div>

                {/* Step 3 */}
                <div className="relative">
                  <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 hover:border-green-500/50 transition-all">
                    <div className="relative inline-block mb-6">
                      <div className="absolute inset-0 bg-green-500 blur-xl opacity-50" />
                      <div className="relative w-16 h-16 bg-green-500 rounded-2xl flex items-center justify-center">
                        <span className="text-3xl font-bold text-black">3</span>
                      </div>
                    </div>
                    <div className="mb-4">
                      <span className="text-7xl font-bold text-gray-800">03</span>
                    </div>
                    <h3 className="text-2xl font-bold mb-3">Journal Trade</h3>
                    <p className="text-gray-400 leading-relaxed">
                      Record your trades and track performance over time
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Trust Signals */}
        <section className="py-20 border-y border-white/5">
          <div className="max-w-[1400px] mx-auto px-8">
            <div className="flex flex-wrap justify-center gap-8">
              <div className="flex items-center space-x-3 px-6 py-3 bg-white/5 rounded-full border border-white/10">
                <div className="w-5 h-5 text-green-400">
                  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                  </svg>
                </div>
                <span className="font-bold text-sm">554 WEEKS ANALYZED</span>
              </div>

              <div className="flex items-center space-x-3 px-6 py-3 bg-white/5 rounded-full border border-white/10">
                <div className="w-5 h-5 text-green-400">
                  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                </div>
                <span className="font-bold text-sm">LIVE VPS DATA</span>
              </div>

              <div className="flex items-center space-x-3 px-6 py-3 bg-white/5 rounded-full border border-white/10">
                <div className="w-5 h-5 text-green-400">
                  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <span className="font-bold text-sm">RULE-DRIVEN</span>
              </div>

              <div className="flex items-center space-x-3 px-6 py-3 bg-white/5 rounded-full border border-white/10">
                <div className="w-5 h-5 text-green-400">
                  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                </div>
                <span className="font-bold text-sm">FOREX READY</span>
              </div>
            </div>
          </div>
        </section>

        {/* Final CTA */}
        <section className="py-40 relative overflow-hidden">
          <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-green-950/30 via-black to-black" />
          
          <div className="max-w-4xl mx-auto px-8 text-center relative z-10">
            <h2 className="text-7xl lg:text-[100px] leading-[0.9] font-bold mb-8 tracking-tight">
              Ready to<br />
              trade <span className="text-green-400 italic font-light">better?</span>
            </h2>
            <p className="text-2xl text-gray-400 mb-12 max-w-2xl mx-auto">
              Join traders who use data and discipline to make smarter decisions
            </p>
            <Link
              href="/login"
              className="inline-flex items-center space-x-3 px-12 py-6 bg-green-500 text-black rounded-2xl font-bold text-xl hover:bg-green-400 transition-all shadow-[0_0_50px_rgba(34,197,94,0.3)]"
            >
              <span>Get Started Free</span>
              <ArrowRight className="w-6 h-6" />
            </Link>
          </div>
        </section>

        {/* Footer */}
        <footer className="border-t border-white/5 bg-gray-950">
          <div className="max-w-[1400px] mx-auto px-8 py-16">
            <div className="grid grid-cols-2 md:grid-cols-5 gap-12 mb-16">
              <div className="col-span-2">
                <div className="flex items-center space-x-3 mb-4">
                  <TrendingUp className="w-6 h-6 text-green-400" />
                  <span className="font-bold text-lg">SET<span className="text-green-400">&</span>TREND</span>
                </div>
                <p className="text-sm text-gray-500 max-w-xs">
                  Rule-driven forex trading platform for systematic decision making
                </p>
              </div>

              <div>
                <h4 className="font-semibold mb-4 text-sm">Product</h4>
                <ul className="space-y-2">
                  <li><Link href="/dashboard" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Markets</Link></li>
                  <li><Link href="/dashboard" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Journal</Link></li>
                  <li><Link href="/dashboard" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Rules</Link></li>
                </ul>
              </div>

              <div>
                <h4 className="font-semibold mb-4 text-sm">Resources</h4>
                <ul className="space-y-2">
                  <li><a href="#docs" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Docs</a></li>
                  <li><a href="#community" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Community</a></li>
                  <li><a href="#api" className="text-sm text-gray-500 hover:text-green-400 transition-colors">API</a></li>
                </ul>
              </div>

              <div>
                <h4 className="font-semibold mb-4 text-sm">Legal</h4>
                <ul className="space-y-2">
                  <li><a href="#privacy" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Privacy</a></li>
                  <li><a href="#terms" className="text-sm text-gray-500 hover:text-green-400 transition-colors">Terms</a></li>
                  <li><a href="https://github.com" className="text-sm text-gray-500 hover:text-green-400 transition-colors">GitHub</a></li>
                </ul>
              </div>
            </div>

            <div className="pt-8 border-t border-white/5 flex flex-col md:flex-row items-center justify-between">
              <p className="text-sm text-gray-600">© 2026 Set and Trend. All rights reserved.</p>
              <p className="text-xs text-gray-700 mt-2 md:mt-0">Built for traders who think in systems</p>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}