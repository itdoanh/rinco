'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/store/auth';
import { RefreshCw, CheckCircle, AlertCircle, XCircle, Activity } from 'lucide-react';

interface ServiceHealth {
  name: string;
  status: 'healthy' | 'degraded' | 'down' | 'unknown';
  uptime: number;
  latency_ms: number;
  last_check: string;
  region?: string;
}

export default function HealthPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [services, setServices] = useState<ServiceHealth[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [lastRefresh, setLastRefresh] = useState(new Date());

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/(auth)/login');
      return;
    }
    fetchHealth();
  }, [isAuthenticated, router]);

  async function fetchHealth() {
    setIsLoading(true);
    try {
      // Demo data
      setServices([
        { name: 'auth-service', status: 'healthy', uptime: 99.99, latency_ms: 12, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'tenant-service', status: 'healthy', uptime: 99.98, latency_ms: 18, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'crm-service', status: 'healthy', uptime: 99.95, latency_ms: 25, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'lead-service', status: 'healthy', uptime: 99.97, latency_ms: 15, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'landing-service', status: 'healthy', uptime: 99.99, latency_ms: 8, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'chat-engine', status: 'healthy', uptime: 99.90, latency_ms: 35, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'webrtc-sfu', status: 'healthy', uptime: 99.85, latency_ms: 45, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'lead-scoring', status: 'healthy', uptime: 99.92, latency_ms: 120, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'ai-sre', status: 'degraded', uptime: 98.50, latency_ms: 250, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'rag-chatbot', status: 'healthy', uptime: 99.80, latency_ms: 180, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'stt-service', status: 'healthy', uptime: 99.75, latency_ms: 200, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'email-service', status: 'healthy', uptime: 99.95, latency_ms: 150, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'notification-service', status: 'healthy', uptime: 99.98, latency_ms: 22, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'postgres-primary', status: 'healthy', uptime: 99.99, latency_ms: 5, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'postgres-replica', status: 'healthy', uptime: 99.98, latency_ms: 6, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'redis-cluster', status: 'healthy', uptime: 99.99, latency_ms: 2, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'nats-server', status: 'healthy', uptime: 99.95, latency_ms: 3, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'minio-storage', status: 'healthy', uptime: 99.90, latency_ms: 15, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'prometheus', status: 'healthy', uptime: 99.85, latency_ms: 10, last_check: new Date().toISOString(), region: 'us-east-1' },
        { name: 'grafana', status: 'unknown', uptime: 0, latency_ms: 0, last_check: new Date().toISOString(), region: 'us-east-1' },
      ]);
      setLastRefresh(new Date());
    } catch (err) {
      console.error('Failed to fetch health:', err);
    } finally {
      setIsLoading(false);
    }
  }

  const getStatusIcon = (status: ServiceHealth['status']) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'degraded':
        return <AlertCircle className="w-5 h-5 text-yellow-400" />;
      case 'down':
        return <XCircle className="w-5 h-5 text-red-400" />;
      default:
        return <Activity className="w-5 h-5 text-gray-400" />;
    }
  };

  const getStatusBadge = (status: ServiceHealth['status']) => {
    const variants = {
      healthy: 'bg-green-500/20 text-green-400 border-green-500/50',
      degraded: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50',
      down: 'bg-red-500/20 text-red-400 border-red-500/50',
      unknown: 'bg-gray-500/20 text-gray-400 border-gray-500/50',
    };
    const labels = {
      healthy: 'Healthy',
      degraded: 'Degraded',
      down: 'Down',
      unknown: 'Unknown',
    };
    return (
      <Badge variant="outline" className={variants[status]}>
        {labels[status]}
      </Badge>
    );
  };

  const healthyCount = services.filter((s) => s.status === 'healthy').length;
  const degradedCount = services.filter((s) => s.status === 'degraded').length;
  const downCount = services.filter((s) => s.status === 'down').length;

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin w-12 h-12 border-4 border-gold border-t-transparent rounded-full"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <nav className="bg-navy-800 border-b border-gray-700 px-6 py-4 flex items-center justify-between">
        <Link href="/dashboard" className="text-2xl font-bold text-gold">RINCO Admin</Link>
        <div className="flex gap-6 items-center">
          <Link href="/dashboard" className="hover:text-gold transition-colors">Dashboard</Link>
          <Link href="/tenants" className="hover:text-gold transition-colors">Tenants</Link>
          <Link href="/system/health" className="text-gold">Health</Link>
        </div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-white">System Health</h1>
            <p className="text-gray-400 mt-1">
              Last checked: {lastRefresh.toLocaleTimeString()}
            </p>
          </div>
          <Button
            onClick={fetchHealth}
            variant="outline"
            className="border-gray-600 text-white hover:bg-navy-700"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh
          </Button>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4 flex items-center gap-4">
              <div className="w-12 h-12 rounded-full bg-green-500/20 flex items-center justify-center">
                <CheckCircle className="w-6 h-6 text-green-400" />
              </div>
              <div>
                <div className="text-2xl font-bold text-white">{healthyCount}</div>
                <div className="text-sm text-gray-400">Healthy</div>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4 flex items-center gap-4">
              <div className="w-12 h-12 rounded-full bg-yellow-500/20 flex items-center justify-center">
                <AlertCircle className="w-6 h-6 text-yellow-400" />
              </div>
              <div>
                <div className="text-2xl font-bold text-white">{degradedCount}</div>
                <div className="text-sm text-gray-400">Degraded</div>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4 flex items-center gap-4">
              <div className="w-12 h-12 rounded-full bg-red-500/20 flex items-center justify-center">
                <XCircle className="w-6 h-6 text-red-400" />
              </div>
              <div>
                <div className="text-2xl font-bold text-white">{downCount}</div>
                <div className="text-sm text-gray-400">Down</div>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4 flex items-center gap-4">
              <div className="w-12 h-12 rounded-full bg-gray-500/20 flex items-center justify-center">
                <Activity className="w-6 h-6 text-gray-400" />
              </div>
              <div>
                <div className="text-2xl font-bold text-white">{services.length}</div>
                <div className="text-sm text-gray-400">Total</div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Services Grid */}
        <Card className="bg-navy-800 border-gray-700">
          <CardHeader>
            <CardTitle className="text-white">Services</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {services.map((service) => (
                <div
                  key={service.name}
                  className="p-4 bg-navy-700 rounded-lg border border-gray-600 hover:border-gray-500 transition-colors"
                >
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <div className="font-medium text-white">{service.name}</div>
                      {service.region && (
                        <div className="text-xs text-gray-500">{service.region}</div>
                      )}
                    </div>
                    {getStatusIcon(service.status)}
                  </div>
                  <div className="flex items-center justify-between">
                    {getStatusBadge(service.status)}
                    {service.status !== 'unknown' && (
                      <div className="text-right">
                        <div className="text-sm text-white">{service.latency_ms}ms</div>
                        <div className="text-xs text-gray-500">{service.uptime}% uptime</div>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
