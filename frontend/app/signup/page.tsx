'use client';

import { useState } from 'react';
import { TrendingUp, ArrowRight } from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { apiClient } from '@/lib/api/client';

export default function SignupPage() {
  const [name, setName] = useState('');
  const [surname, setSurname] = useState('');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const response = await apiClient.signup({
        username,
        email,
        password,
        name: name || undefined,
        surname: surname || undefined,
      });

      // Save token to localStorage
      localStorage.setItem('auth_token', response.data.token);
      localStorage.setItem('user', JSON.stringify(response.data.user));

      // Redirect to dashboard
      router.push('/dashboard');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Signup failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white flex items-center justify-center relative overflow-hidden">
      {/* Background Grid */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff08_1px,transparent_1px),linear-gradient(to_bottom,#ffffff08_1px,transparent_1px)] bg-[size:64px_64px]" />
      
      {/* Green Glow */}
      <div className="absolute top-1/3 left-1/2 w-[500px] h-[500px] bg-green-500/20 rounded-full blur-[120px] -translate-x-1/2" />

      <div className="relative z-10 w-full max-w-md px-8 py-12">
        {/* Logo */}
        <Link href="/" className="flex items-center justify-center space-x-3 mb-12 group">
          <div className="relative">
            <div className="absolute inset-0 bg-green-500 blur-lg opacity-50 group-hover:opacity-70 transition-opacity" />
            <TrendingUp className="w-8 h-8 text-green-400 relative" />
          </div>
          <span className="text-2xl font-bold tracking-tight">
            SET<span className="text-green-400">&</span>TREND
          </span>
        </Link>

        {/* Signup Card */}
        <div className="bg-gradient-to-br from-gray-900 to-black border border-white/10 rounded-2xl p-8 shadow-2xl">
          <h1 className="text-3xl font-bold mb-2">Create account</h1>
          <p className="text-gray-400 mb-8">Start your systematic trading journey</p>

          {error && (
            <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 mb-4">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Name Field */}
            <div>
              <label htmlFor="name" className="block text-sm font-medium mb-2">
                Name
              </label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-lg focus:outline-none focus:border-green-500 transition-colors text-white"
                placeholder="Enter your name"
                required
              />
            </div>

            {/* Surname Field */}
            <div>
              <label htmlFor="surname" className="block text-sm font-medium mb-2">
                Surname
              </label>
              <input
                id="surname"
                type="text"
                value={surname}
                onChange={(e) => setSurname(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-lg focus:outline-none focus:border-green-500 transition-colors text-white"
                placeholder="Enter your surname"
                required
              />
            </div>

            {/* Username Field */}
            <div>
              <label htmlFor="username" className="block text-sm font-medium mb-2">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-lg focus:outline-none focus:border-green-500 transition-colors text-white"
                placeholder="Choose a username"
                required
              />
            </div>

            {/* Email Field */}
            <div>
              <label htmlFor="email" className="block text-sm font-medium mb-2">
                Email
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-lg focus:outline-none focus:border-green-500 transition-colors text-white"
                placeholder="Enter your email"
                required
              />
            </div>

            {/* Password Field */}
            <div>
              <label htmlFor="password" className="block text-sm font-medium mb-2">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 bg-black/50 border border-white/10 rounded-lg focus:outline-none focus:border-green-500 transition-colors text-white"
                placeholder="Create a password"
                required
              />
            </div>

            {/* Terms & Conditions */}
            <div className="flex items-start space-x-2 text-sm">
              <input 
                type="checkbox" 
                id="terms"
                className="w-4 h-4 rounded border-white/10 bg-black/50 mt-0.5" 
                required
              />
              <label htmlFor="terms" className="text-gray-400">
                I agree to the{' '}
                <a href="#terms" className="text-green-400 hover:text-green-300 transition-colors">
                  Terms of Service
                </a>
                {' '}and{' '}
                <a href="#privacy" className="text-green-400 hover:text-green-300 transition-colors">
                  Privacy Policy
                </a>
              </label>
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-green-500 text-black rounded-lg font-semibold hover:bg-green-400 transition-all flex items-center justify-center space-x-2 shadow-lg shadow-green-500/30 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <span>{loading ? 'Creating Account...' : 'Create Account'}</span>
              {!loading && <ArrowRight className="w-5 h-5" />}
            </button>
          </form>

          {/* Login Link */}
          <div className="mt-6 text-center text-sm">
            <span className="text-gray-400">Already have an account? </span>
            <Link href="/login" className="text-green-400 hover:text-green-300 transition-colors font-medium">
              Sign in
            </Link>
          </div>
        </div>

        {/* Back to Home */}
        <div className="mt-8 text-center">
          <Link href="/" className="text-sm text-gray-500 hover:text-gray-400 transition-colors">
            ← Back to home
          </Link>
        </div>
      </div>
    </div>
  );
}
