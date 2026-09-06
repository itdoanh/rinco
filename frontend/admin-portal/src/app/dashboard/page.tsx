'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';

interface DashboardStats {
  tenants: number;
  users: number;
  leads: number;
  messages24h: number;
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats>({
    tenants: 0,
    users: 0,
    leads: 0,
    messages24h: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (!localStorage.getItem('access_token')) {
      window.location.href = '/';
      return;
    }

    fetchStats();
  }, []);

  async function fetchStats() {
    try {
      const token = localStorage.getItem('access_token');
      const headers = { Authorization: `Bearer ${token}` };

      // In production: call real APIs
      // For demo: mock data
      setStats({
        tenants: 12,
        users: 248,
        leads: 1820,
        messages24h: 9500,
      });
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin w-12 h-12 border-4 border-gold border-t-transparent rounded-full"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <nav className="bg-navy-800 border-b border-gray-700 px-6 py-4 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gold">RINCO Admin</h1>
        <div className="flex gap-6 items-center">
          <Link href="/dashboard" className="hover:text-gold">Dashboard</Link>
          <Link href="/tenants" className="hover:text-gold">Tenants</Link>
          <Link href="/users" className="hover:text-gold">Users</Link>
          <Link href="/leads" className="hover:text-gold">Leads</Link>
          <button onClick={() => { localStorage.clear(); window.location.href = '/'; }} className="text-red-400">Logout</button>
        </div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        <h2 className="text-3xl font-bold mb-6">Dashboard</h2>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard label="Tenants" value={stats.tenants} icon="🏢" />
          <StatCard label="Users" value={stats.users} icon="👥" />
          <StatCard label="Leads" value={stats.leads} icon="🎯" />
          <StatCard label="Messages (24h)" value={stats.messages24h} icon="💬" />
        </div>

        <div className="grid lg:grid-cols-2 gap-6">
          <div className="bg-navy-800 rounded-2xl p-6 border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Service Health</h3>
            <ServiceHealthList />
          </div>

          <div className="bg-navy-800 rounded-2xl p-6 border border-gray-700">
            <h3 className="text-xl font-bold mb-4">Recent Activity</h3>
            <div className="space-y-3 text-sm text-gray-400">
              <p>• New tenant created: <span className="text-white">acme-corp</span></p>
              <p>• Lead scored hot: <span className="text-gold">Nguyễn Văn A</span></p>
              <p>• FIDO2 enrollment: <span className="text-white">5 users</span></p>
              <p>• Alert resolved: <span className="text-green-400">DB latency</span></p>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

function StatCard({ label, value, icon }: { label: string; value: number; icon: string }) {
  return (
    <div className="bg-navy-800 rounded-2xl p-6 border border-gray-700 hover:border-gold transition-colors">
      <div className="text-4xl mb-3">{icon}</div>
      <div className="text-3xl font-bold text-gold">{value.toLocaleString()}</div>
      <div className="text-sm text-gray-400 mt-1">{label}</div>
    </div>
  );
}

function ServiceHealthList() {
  const services = [
    { name: 'auth-service', status: 'ok' },
    { name: 'tenant-service', status: 'ok' },
    { name: 'crm-service', status: 'ok' },
    { name: 'dynamic-model-service', status: 'ok' },
    { name: 'lead-service', status: 'ok' },
    { name: 'landing-service', status: 'ok' },
    { name: 'chat-engine', status: 'ok' },
    { name: 'webrtc-sfu', status: 'ok' },
    { name: 'lead-scoring', status: 'ok' },
    { name: 'ai-sre', status: 'ok' },
    { name: 'rag-chatbot', status: 'ok' },
    { name: 'stt-service', status: 'ok' },
    { name: 'email-service', status: 'ok' },
    { name: 'notification-service', status: 'ok' },
  ];
  return (
    <div className="grid grid-cols-2 gap-2 text-sm">
      {services.map((s) => (
        <div key={s.name} className="flex items-center gap-2">
          <span className={s.status === 'ok' ? 'text-green-400' : 'text-red-400'}>●</span>
          <span>{s.name}</span>
        </div>
      ))}
    </div>
  );
}
