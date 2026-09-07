'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { use } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ArrowLeft, Edit2, Settings, Users, BarChart3, ExternalLink } from 'lucide-react';
import { useAuthStore } from '@/store/auth';

interface TenantDetail {
  id: string;
  name: string;
  slug: string;
  status: 'active' | 'inactive' | 'suspended';
  plan: 'free' | 'starter' | 'professional' | 'enterprise';
  domain?: string;
  created_at: string;
  updated_at: string;
  owner: {
    name: string;
    email: string;
  };
  stats: {
    total_users: number;
    active_users: number;
    total_leads: number;
    total_pages: number;
    api_calls_30d: number;
  };
  users: Array<{
    id: string;
    name: string;
    email: string;
    role: string;
    last_login: string;
  }>;
}

export default function TenantDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [tenant, setTenant] = useState<TenantDetail | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/(auth)/login');
      return;
    }
    fetchTenantDetail();
  }, [isAuthenticated, id, router]);

  async function fetchTenantDetail() {
    try {
      // Demo data
      setTenant({
        id,
        name: 'ACME Corporation',
        slug: 'acme-corp',
        status: 'active',
        plan: 'professional',
        domain: 'acme.example.com',
        created_at: '2024-01-15',
        updated_at: '2024-09-01',
        owner: {
          name: 'Nguyễn Văn A',
          email: 'admin@acme.com',
        },
        stats: {
          total_users: 45,
          active_users: 38,
          total_leads: 1250,
          total_pages: 8,
          api_calls_30d: 45000,
        },
        users: [
          { id: '1', name: 'Nguyễn Văn A', email: 'admin@acme.com', role: 'Admin', last_login: '2024-09-07' },
          { id: '2', name: 'Trần Thị B', email: 'trangthai@acme.com', role: 'Manager', last_login: '2024-09-06' },
          { id: '3', name: 'Lê Văn C', email: 'levanc@acme.com', role: 'User', last_login: '2024-09-05' },
        ],
      });
    } catch (err) {
      console.error('Failed to fetch tenant:', err);
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

  if (!tenant) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white mb-2">Tenant không tìm thấy</h1>
          <Link href="/tenants">
            <Button variant="outline" className="border-gray-600 text-white hover:bg-navy-700">
              Quay lại
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  const statusColors = {
    active: 'bg-green-500/20 text-green-400 border-green-500/50',
    inactive: 'bg-gray-500/20 text-gray-400 border-gray-500/50',
    suspended: 'bg-red-500/20 text-red-400 border-red-500/50',
  };

  const planColors = {
    free: 'bg-gray-500/20 text-gray-400',
    starter: 'bg-blue-500/20 text-blue-400',
    professional: 'bg-purple-500/20 text-purple-400',
    enterprise: 'bg-gold/20 text-gold',
  };

  return (
    <div className="min-h-screen">
      <nav className="bg-navy-800 border-b border-gray-700 px-6 py-4 flex items-center gap-4">
        <Link href="/tenants">
          <Button variant="ghost" size="sm" className="hover:text-gold">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Quay lại
          </Button>
        </Link>
        <div className="text-xl font-bold text-gold">RINCO Admin</div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-start justify-between mb-6">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-bold text-white">{tenant.name}</h1>
              <Badge variant="outline" className={statusColors[tenant.status]}>
                {tenant.status === 'active' ? 'Hoạt động' : tenant.status === 'inactive' ? 'Không hoạt động' : 'Đình chỉ'}
              </Badge>
              <Badge className={planColors[tenant.plan]}>
                {tenant.plan.charAt(0).toUpperCase() + tenant.plan.slice(1)}
              </Badge>
            </div>
            <div className="text-gray-500">
              {tenant.slug} • {tenant.domain || 'Không có domain'}
            </div>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" className="border-gray-600 text-white hover:bg-navy-700">
              <Settings className="w-4 h-4 mr-2" />
              Cài đặt
            </Button>
            <Button variant="outline" className="border-gray-600 text-white hover:bg-navy-700">
              <Edit2 className="w-4 h-4 mr-2" />
              Chỉnh sửa
            </Button>
            {tenant.domain && (
              <Button className="bg-gold hover:bg-gold-light text-navy-900">
                <ExternalLink className="w-4 h-4 mr-2" />
                Xem website
              </Button>
            )}
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="text-sm text-gray-400 mb-1">Tổng Users</div>
              <div className="text-2xl font-bold text-white">{tenant.stats.total_users}</div>
            </CardContent>
          </Card>
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="text-sm text-gray-400 mb-1">Active Users</div>
              <div className="text-2xl font-bold text-green-400">{tenant.stats.active_users}</div>
            </CardContent>
          </Card>
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="text-sm text-gray-400 mb-1">Tổng Leads</div>
              <div className="text-2xl font-bold text-white">{tenant.stats.total_leads.toLocaleString()}</div>
            </CardContent>
          </Card>
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="text-sm text-gray-400 mb-1">Landing Pages</div>
              <div className="text-2xl font-bold text-white">{tenant.stats.total_pages}</div>
            </CardContent>
          </Card>
          <Card className="bg-navy-800 border-gray-700">
            <CardContent className="p-4">
              <div className="text-sm text-gray-400 mb-1">API Calls (30d)</div>
              <div className="text-2xl font-bold text-white">{(tenant.stats.api_calls_30d / 1000).toFixed(1)}K</div>
            </CardContent>
          </Card>
        </div>

        {/* Tabs */}
        <Tabs defaultValue="users" className="space-y-4">
          <TabsList className="bg-navy-800 border-gray-700">
            <TabsTrigger value="users" className="data-[state=active]:bg-gold data-[state=active]:text-navy-900">
              <Users className="w-4 h-4 mr-2" />
              Users
            </TabsTrigger>
            <TabsTrigger value="pages" className="data-[state=active]:bg-gold data-[state=active]:text-navy-900">
              <BarChart3 className="w-4 h-4 mr-2" />
              Pages
            </TabsTrigger>
            <TabsTrigger value="settings" className="data-[state=active]:bg-gold data-[state=active]:text-navy-900">
              <Settings className="w-4 h-4 mr-2" />
              Settings
            </TabsTrigger>
          </TabsList>

          <TabsContent value="users">
            <Card className="bg-navy-800 border-gray-700">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>Danh sách Users</CardTitle>
                  <Button size="sm" className="bg-gold hover:bg-gold-light text-navy-900">
                    Thêm User
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="p-0">
                <Table>
                  <TableHeader>
                    <TableRow className="border-gray-700">
                      <TableHead className="text-gray-400">Name</TableHead>
                      <TableHead className="text-gray-400">Email</TableHead>
                      <TableHead className="text-gray-400">Role</TableHead>
                      <TableHead className="text-gray-400">Last Login</TableHead>
                      <TableHead className="text-gray-400 text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tenant.users.map((user) => (
                      <TableRow key={user.id} className="border-gray-700">
                        <TableCell className="font-medium text-white">{user.name}</TableCell>
                        <TableCell className="text-gray-400">{user.email}</TableCell>
                        <TableCell>
                          <Badge variant="outline" className="border-gray-600 text-gray-400">
                            {user.role}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-gray-400">{user.last_login}</TableCell>
                        <TableCell className="text-right">
                          <Button size="sm" variant="ghost" className="hover:text-gold">
                            <Edit2 className="w-4 h-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="pages">
            <Card className="bg-navy-800 border-gray-700">
              <CardHeader>
                <CardTitle>Landing Pages</CardTitle>
                <CardDescription>Danh sách các trang landing của tenant</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-gray-500">Chưa có dữ liệu pages</p>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="settings">
            <Card className="bg-navy-800 border-gray-700">
              <CardHeader>
                <CardTitle>Cài đặt Tenant</CardTitle>
                <CardDescription>Quản lý cấu hình tenant</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <Label className="text-gray-400">Tên Tenant</Label>
                    <Input defaultValue={tenant.name} className="bg-navy-700 border-gray-600 text-white mt-1" />
                  </div>
                  <div>
                    <Label className="text-gray-400">Slug</Label>
                    <Input defaultValue={tenant.slug} className="bg-navy-700 border-gray-600 text-white mt-1" />
                  </div>
                  <div>
                    <Label className="text-gray-400">Domain</Label>
                    <Input defaultValue={tenant.domain || ''} placeholder="https://..." className="bg-navy-700 border-gray-600 text-white mt-1" />
                  </div>
                  <div>
                    <Label className="text-gray-400">Plan</Label>
                    <Input defaultValue={tenant.plan} disabled className="bg-navy-600 border-gray-600 text-gray-400 mt-1" />
                  </div>
                </div>
                <div className="flex justify-end">
                  <Button className="bg-gold hover:bg-gold-light text-navy-900">
                    Lưu thay đổi
                  </Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </main>
    </div>
  );
}
