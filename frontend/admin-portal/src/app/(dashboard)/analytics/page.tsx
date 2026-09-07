'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { TrendingUp, TrendingDown, Users, Target, DollarSign, Activity } from 'lucide-react';
import { useAuthStore } from '@/store/auth';

interface AnalyticsData {
  overview: {
    total_leads: number;
    leads_growth: number;
    conversion_rate: number;
    conversion_growth: number;
    revenue: number;
    revenue_growth: number;
    active_users: number;
    users_growth: number;
  };
  leads_trend: Array<{ date: string; leads: number; conversions: number }>;
  traffic_sources: Array<{ name: string; value: number; color: string }>;
  top_pages: Array<{ page: string; views: number; leads: number; conversion: number }>;
  conversions_by_day: Array<{ day: string; rate: number }>;
}

const COLORS = ['#F5A623', '#3B82F6', '#10B981', '#EF4444', '#8B5CF6', '#EC4899'];

export default function AnalyticsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [dateRange, setDateRange] = useState('7d');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/(auth)/login');
      return;
    }
    fetchAnalytics();
  }, [isAuthenticated, router, dateRange]);

  async function fetchAnalytics() {
    try {
      // Demo data
      setData({
        overview: {
          total_leads: 1247,
          leads_growth: 12.5,
          conversion_rate: 4.2,
          conversion_growth: 0.8,
          revenue: 185000000,
          revenue_growth: 8.3,
          active_users: 342,
          users_growth: 5.2,
        },
        leads_trend: [
          { date: '2024-09-01', leads: 120, conversions: 5 },
          { date: '2024-09-02', leads: 145, conversions: 6 },
          { date: '2024-09-03', leads: 132, conversions: 7 },
          { date: '2024-09-04', leads: 168, conversions: 8 },
          { date: '2024-09-05', leads: 155, conversions: 6 },
          { date: '2024-09-06', leads: 178, conversions: 9 },
          { date: '2024-09-07', leads: 195, conversions: 10 },
        ],
        traffic_sources: [
          { name: 'Direct', value: 35, color: '#F5A623' },
          { name: 'Organic Search', value: 28, color: '#3B82F6' },
          { name: 'Paid Ads', value: 22, color: '#10B981' },
          { name: 'Social Media', value: 10, color: '#EF4444' },
          { name: 'Referral', value: 5, color: '#8B5CF6' },
        ],
        top_pages: [
          { page: '/san-pham/chuyen-doi-so', views: 15420, leads: 680, conversion: 4.4 },
          { page: '/dich-vu/consulting', views: 12350, leads: 420, conversion: 3.4 },
          { page: '/blog/seo-huong-dan', views: 9800, leads: 85, conversion: 0.9 },
          { page: '/ve-chung-toi', views: 6500, leads: 45, conversion: 0.7 },
          { page: '/lien-he', views: 4200, leads: 120, conversion: 2.9 },
        ],
        conversions_by_day: [
          { day: 'Mon', rate: 3.8 },
          { day: 'Tue', rate: 4.2 },
          { day: 'Wed', rate: 4.5 },
          { day: 'Thu', rate: 4.1 },
          { day: 'Fri', rate: 3.9 },
          { day: 'Sat', rate: 2.8 },
          { day: 'Sun', rate: 2.5 },
        ],
      });
    } catch (err) {
      console.error('Failed to fetch analytics:', err);
    } finally {
      setIsLoading(false);
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin w-12 h-12 border-4 border-gold border-t-transparent rounded-full"></div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-white">Không thể tải dữ liệu analytics</div>
      </div>
    );
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency: 'VND',
      minimumFractionDigits: 0,
    }).format(value);
  };

  const formatNumber = (value: number) => {
    return new Intl.NumberFormat('vi-VN').format(value);
  };

  return (
    <div className="min-h-screen">
      <nav className="bg-navy-800 border-b border-gray-700 px-6 py-4 flex items-center justify-between">
        <Link href="/dashboard" className="text-2xl font-bold text-gold">RINCO Admin</Link>
        <div className="flex gap-6 items-center">
          <Link href="/dashboard" className="hover:text-gold transition-colors">Dashboard</Link>
          <Link href="/tenants" className="hover:text-gold transition-colors">Tenants</Link>
          <Link href="/analytics" className="text-gold">Analytics</Link>
          <Link href="/users" className="hover:text-gold transition-colors">Users</Link>
        </div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold text-white">Analytics Dashboard</h1>
          <div className="flex gap-2">
            {['24h', '7d', '30d', '90d'].map((range) => (
              <button
                key={range}
                onClick={() => setDateRange(range)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  dateRange === range
                    ? 'bg-gold text-navy-900'
                    : 'bg-navy-800 text-gray-400 hover:bg-navy-700'
                }`}
              >
                {range === '24h' ? '24 giờ' : range === '7d' ? '7 ngày' : range === '30d' ? '30 ngày' : '90 ngày'}
              </button>
            ))}
          </div>
        </div>

        {/* Overview Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">Tổng Leads</span>
                <Target className="w-5 h-5 text-gold" />
              </div>
              <div className="text-3xl font-bold text-white mb-1">{formatNumber(data.overview.total_leads)}</div>
              <div className="flex items-center text-sm">
                {data.overview.leads_growth > 0 ? (
                  <TrendingUp className="w-4 h-4 text-green-400 mr-1" />
                ) : (
                  <TrendingDown className="w-4 h-4 text-red-400 mr-1" />
                )}
                <span className={data.overview.leads_growth > 0 ? 'text-green-400' : 'text-red-400'}>
                  {data.overview.leads_growth > 0 ? '+' : ''}{data.overview.leads_growth}%
                </span>
                <span className="text-gray-500 ml-1">vs last period</span>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">Conversion Rate</span>
                <Activity className="w-5 h-5 text-purple-400" />
              </div>
              <div className="text-3xl font-bold text-white mb-1">{data.overview.conversion_rate}%</div>
              <div className="flex items-center text-sm">
                {data.overview.conversion_growth > 0 ? (
                  <TrendingUp className="w-4 h-4 text-green-400 mr-1" />
                ) : (
                  <TrendingDown className="w-4 h-4 text-red-400 mr-1" />
                )}
                <span className={data.overview.conversion_growth > 0 ? 'text-green-400' : 'text-red-400'}>
                  {data.overview.conversion_growth > 0 ? '+' : ''}{data.overview.conversion_growth}%
                </span>
                <span className="text-gray-500 ml-1">vs last period</span>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">Revenue</span>
                <DollarSign className="w-5 h-5 text-green-400" />
              </div>
              <div className="text-3xl font-bold text-white mb-1">{formatCurrency(data.overview.revenue)}</div>
              <div className="flex items-center text-sm">
                {data.overview.revenue_growth > 0 ? (
                  <TrendingUp className="w-4 h-4 text-green-400 mr-1" />
                ) : (
                  <TrendingDown className="w-4 h-4 text-red-400 mr-1" />
                )}
                <span className={data.overview.revenue_growth > 0 ? 'text-green-400' : 'text-red-400'}>
                  {data.overview.revenue_growth > 0 ? '+' : ''}{data.overview.revenue_growth}%
                </span>
                <span className="text-gray-500 ml-1">vs last period</span>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-400">Active Users</span>
                <Users className="w-5 h-5 text-blue-400" />
              </div>
              <div className="text-3xl font-bold text-white mb-1">{formatNumber(data.overview.active_users)}</div>
              <div className="flex items-center text-sm">
                {data.overview.users_growth > 0 ? (
                  <TrendingUp className="w-4 h-4 text-green-400 mr-1" />
                ) : (
                  <TrendingDown className="w-4 h-4 text-red-400 mr-1" />
                )}
                <span className={data.overview.users_growth > 0 ? 'text-green-400' : 'text-red-400'}>
                  {data.overview.users_growth > 0 ? '+' : ''}{data.overview.users_growth}%
                </span>
                <span className="text-gray-500 ml-1">vs last period</span>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <Card className="bg-navy-800 border-gray-700">
            <CardHeader>
              <CardTitle className="text-white">Leads Trend</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={data.leads_trend}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="date" stroke="#9CA3AF" fontSize={12} />
                  <YAxis stroke="#9CA3AF" fontSize={12} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', border: '1px solid #374151', borderRadius: '8px' }}
                    labelStyle={{ color: '#F3F4F6' }}
                  />
                  <Legend />
                  <Line type="monotone" dataKey="leads" stroke="#F5A623" strokeWidth={2} name="Leads" />
                  <Line type="monotone" dataKey="conversions" stroke="#10B981" strokeWidth={2" name="Conversions" />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardHeader>
              <CardTitle className="text-white">Traffic Sources</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={data.traffic_sources}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    outerRadius={100}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  >
                    {data.traffic_sources.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', border: '1px solid #374151', borderRadius: '8px' }}
                    labelStyle={{ color: '#F3F4F6' }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card className="bg-navy-800 border-gray-700">
            <CardHeader>
              <CardTitle className="text-white">Top Pages</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {data.top_pages.map((page, index) => (
                  <div key={page.page} className="flex items-center justify-between p-3 bg-navy-700 rounded-lg">
                    <div>
                      <div className="text-white font-medium">{page.page}</div>
                      <div className="text-sm text-gray-400">
                        {formatNumber(page.views)} views • {formatNumber(page.leads)} leads
                      </div>
                    </div>
                    <Badge className="bg-green-500/20 text-green-400">
                      {page.conversion}% CVR
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardHeader>
              <CardTitle className="text-white">Conversion Rate by Day</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={data.conversions_by_day}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="day" stroke="#9CA3AF" fontSize={12} />
                  <YAxis stroke="#9CA3AF" fontSize={12} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#1F2937', border: '1px solid #374151', borderRadius: '8px' }}
                    labelStyle={{ color: '#F3F4F6' }}
                  />
                  <Bar dataKey="rate" fill="#F5A623" name="CVR %" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
