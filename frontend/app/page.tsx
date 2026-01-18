'use client';

import Link from 'next/link';
import { ArrowRight, TrendingUp } from 'lucide-react';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#0a0a0a] text-white">
      {/* Fixed Navbar */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-[#0a0a0a]/95 backdrop-blur-sm border-b border-gray-800">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          {/* Logo */}
          <Link href="/" className="flex items-center space-x-2">
            <TrendingUp className="w-8 h-8 text-green-500" />
            <span className="text-2xl font-bold bg-gradient-to-r from-green-400 to-emerald-500 bg-clip-text text-transparent">
              SET AND TREND
            </span>
          </Link>

          {/* Right CTAs */}
          <div className="flex items-center space-x-4">
            <Link
              href="/signup"
              className="px-6 py-2 border border-green-500/50 text-green-400 rounded-lg hover:bg-green-500/10 transition-all"
            >
              Sign Up Free
            </Link>
            <Link
              href="/login"
              className="px-6 py-2 bg-gradient-to-r from-green-500 to-emerald-600 rounded-lg hover:from-green-600 hover:to-emerald-700 transition-all font-semibold"
            >
              Log In
            </Link>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <div className="pt-20">
        {/* Hero Section */}
        <section className="min-h-screen flex items-center">
          <div className="max-w-7xl mx-auto px-6 grid grid-cols-1 lg:grid-cols-2 gap-12">
            {/* Left: Text */}
            <div className="flex flex-col justify-center space-y-8">
              <h1 className="text-6xl font-bold leading-tight">
                MASTER FOREX TRENDS WITH{' '}
                <span className="bg-gradient-to-r from-green-400 to-emerald-500 bg-clip-text text-transparent">
                  RULE-DRIVEN DISCIPLINE
                </span>
              </h1>
              
              <p className="text-xl text-gray-400 leading-relaxed">
                Weekly charts + EMA signals + PASS/FAIL rules across all pairs.
              </p>

              <div className="flex space-x-4">
                <Link
                  href="/signup"
                  className="px-8 py-4 bg-gradient-to-r from-green-500 to-emerald-600 rounded-lg hover:from-green-600 hover:to-emerald-700 transition-all font-semibold text-lg flex items-center space-x-2"
                >
                  <span>Get Started Free</span>
                  <ArrowRight className="w-5 h-5" />
                </Link>
                <Link
                  href="/dashboard"
                  className="px-8 py-4 border border-gray-700 rounded-lg hover:bg-gray-800 transition-all font-semibold text-lg"
                >
                  Live Demo
                </Link>
              </div>
            </div>

            {/* Right: Dashboard Screenshot - BOX STRETCHES TO MATCH LEFT */}
            <div className="relative flex items-center">
              {/* Glow behind the box */}
              <div className="absolute -inset-4 bg-gradient-to-r from-green-500/20 to-emerald-500/20 blur-3xl" />
              
              {/* Border box - NOW FILLS FULL HEIGHT */}
              <div className="relative w-full h-full border border-green-500/30 rounded-xl overflow-hidden shadow-2xl bg-gray-900 flex flex-col">
                <div className="p-6 flex-1 flex flex-col justify-between">
                  {/* Chart takes most space */}
                  <div className="flex-1 rounded-lg border border-gray-700 overflow-hidden mb-4">
                    <img 
                      src="/weekly_chart.png" 
                      alt="EURUSD Weekly Chart with EMA indicators" 
                      className="w-full h-full object-cover"
                    />
                  </div>
                  
                  {/* Stats at bottom */}
                  <div className="grid grid-cols-3 gap-3">
                    <div className="h-20 bg-gray-800 rounded border border-gray-700 flex flex-col items-center justify-center">
                      <p className="text-xs text-gray-500 uppercase">EMA 50</p>
                      <p className="text-green-400 font-semibold text-lg">↑ BULL</p>
                    </div>
                    <div className="h-20 bg-gray-800 rounded border border-gray-700 flex flex-col items-center justify-center">
                      <p className="text-xs text-gray-500 uppercase">Range</p>
                      <p className="text-white font-semibold text-lg">154 pips</p>
                    </div>
                    <div className="h-20 bg-gray-800 rounded border border-gray-700 flex flex-col items-center justify-center">
                      <p className="text-xs text-gray-500 uppercase">Signal</p>
                      <p className="text-green-400 font-semibold text-lg">PASS</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* More sections coming next... */}
      </div>
    </div>
  );
}