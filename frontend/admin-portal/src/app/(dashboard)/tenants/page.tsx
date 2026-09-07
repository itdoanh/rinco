'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Plus, Search, Edit2, Trash2, Eye } from 'lucide-react';
import { useAuthStore } from '@/store/auth';

interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: 'active' | 'inactive' | 'suspended';
  plan: 'free' | 'starter' | 'professional' | 'enterprise';
  users_count: number;
  created_at: string;
  domain?: string;
}

export default function TenantsPage() {
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/(auth)/login');
      return;
    }
    fetchTenants();
  }, [isAuthenticated, router]);

  async function fetchTenants() {
    try {
      const token = localStorage.getItem('access_token');
      // Demo data
      setTenants([
        {
          id: '1',
          name: 'ACME Corporation',
          slug: 'acme-corp',
          status: 'active',
          plan: 'professional',
          users_count: 45,
          created_at: '2024-01-15',
          domain: 'acme.example.com',
        },
        {
          id: '2',
          name: 'TechStart Vietnam',
          slug: 'techstart',
          status: 'active',
          plan: 'starter',
          users_count: 12,
          created_at: '2024-03-22',
          domain: 'techstart.vn',
        },
        {
          id: '3',
          name: 'Global Finance',
          slug: 'global-finance',
          status: 'inactive',
          plan: 'enterprise',
          users_count: 120,
          created_at: '2023-11-08',
        },
        {
          id: '4',
          name: 'EduLearn Platform',
          slug: 'edulearn',
          status: 'active',
          plan: 'professional',
          users_count: 28,
          created_at: '2024-05-10',
          domain: 'edulearn.edu.vn',
        },
      ]);
    } catch (err) {
      console.error('Failed to fetch tenants:', err);
    } finally {
      setIsLoading(false);
    }
  }

  const filteredTenants = tenants.filter((tenant) => {
    const matchesSearch =
      tenant.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tenant.slug.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesStatus = statusFilter === 'all' || tenant.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const totalPages = Math.ceil(filteredTenants.length / itemsPerPage);
  const paginatedTenants = filteredTenants.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const getStatusBadge = (status: Tenant['status']) => {
    const variants = {
      active: 'bg-green-500/20 text-green-400 border-green-500/50',
      inactive: 'bg-gray-500/20 text-gray-400 border-gray-500/50',
      suspended: 'bg-red-500/20 text-red-400 border-red-500/50',
    };
    const labels = {
      active: 'Hoạt động',
      inactive: 'Không hoạt động',
      suspended: 'Đình chỉ',
    };
    return (
      <Badge variant="outline" className={variants[status]}>
        {labels[status]}
      </Badge>
    );
  };

  const getPlanBadge = (plan: Tenant['plan']) => {
    const variants = {
      free: 'bg-gray-500/20 text-gray-400',
      starter: 'bg-blue-500/20 text-blue-400',
      professional: 'bg-purple-500/20 text-purple-400',
      enterprise: 'bg-gold/20 text-gold',
    };
    const labels = {
      free: 'Free',
      starter: 'Starter',
      professional: 'Professional',
      enterprise: 'Enterprise',
    };
    return (
      <Badge className={variants[plan]}>
        {labels[plan]}
      </Badge>
    );
  };

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
          <Link href="/tenants" className="text-gold">Tenants</Link>
          <Link href="/users" className="hover:text-gold transition-colors">Users</Link>
          <button onClick={() => { localStorage.clear(); router.push('/(auth)/login'); }} className="text-red-400">Logout</button>
        </div>
      </nav>

      <main className="p-6 max-w-7xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold">Quản lý Tenants</h1>
          <Button className="bg-gold hover:bg-gold-light text-navy-900">
            <Plus className="w-4 h-4 mr-2" />
            Thêm Tenant
          </Button>
        </div>

        <Card className="bg-navy-800 border-gray-700 mb-6">
          <CardContent className="p-4">
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
                <Input
                  placeholder="Tìm kiếm tenant..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 bg-navy-700 border-gray-600 text-white"
                />
              </div>
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-full sm:w-48 bg-navy-700 border-gray-600 text-white">
                  <SelectValue placeholder="Trạng thái" />
                </SelectTrigger>
                <SelectContent className="bg-navy-700 border-gray-600 text-white">
                  <SelectItem value="all">Tất cả</SelectItem>
                  <SelectItem value="active">Hoạt động</SelectItem>
                  <SelectItem value="inactive">Không hoạt động</SelectItem>
                  <SelectItem value="suspended">Đình chỉ</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-navy-800 border-gray-700">
          <CardHeader>
            <CardTitle className="text-lg">Danh sách Tenants ({filteredTenants.length})</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="border-gray-700 hover:bg-navy-700/50">
                    <TableHead className="text-gray-400">Tên</TableHead>
                    <TableHead className="text-gray-400">Slug</TableHead>
                    <TableHead className="text-gray-400">Trạng thái</TableHead>
                    <TableHead className="text-gray-400">Plan</TableHead>
                    <TableHead className="text-gray-400">Users</TableHead>
                    <TableHead className="text-gray-400">Ngày tạo</TableHead>
                    <TableHead className="text-gray-400 text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedTenants.map((tenant) => (
                    <TableRow key={tenant.id} className="border-gray-700 hover:bg-navy-700/50">
                      <TableCell className="font-medium text-white">
                        <div>
                          <div>{tenant.name}</div>
                          {tenant.domain && (
                            <div className="text-sm text-gray-500">{tenant.domain}</div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-gray-400">{tenant.slug}</TableCell>
                      <TableCell>{getStatusBadge(tenant.status)}</TableCell>
                      <TableCell>{getPlanBadge(tenant.plan)}</TableCell>
                      <TableCell className="text-gray-400">{tenant.users_count}</TableCell>
                      <TableCell className="text-gray-400">{tenant.created_at}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Link href={`/tenants/${tenant.id}`}>
                            <Button size="sm" variant="ghost" className="hover:text-gold">
                              <Eye className="w-4 h-4" />
                            </Button>
                          </Link>
                          <Button size="sm" variant="ghost" className="hover:text-gold">
                            <Edit2 className="w-4 h-4" />
                          </Button>
                          <Button size="sm" variant="ghost" className="hover:text-red-400">
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {filteredTenants.length === 0 && (
              <div className="text-center py-12 text-gray-500">
                Không tìm thấy tenant nào
              </div>
            )}

            {totalPages > 1 && (
              <div className="flex items-center justify-between p-4 border-t border-gray-700">
                <div className="text-sm text-gray-500">
                  Trang {currentPage} / {totalPages}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                    disabled={currentPage === 1}
                    className="border-gray-600 text-white hover:bg-navy-700"
                  >
                    Trước
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                    disabled={currentPage === totalPages}
                    className="border-gray-600 text-white hover:bg-navy-700"
                  >
                    Sau
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
