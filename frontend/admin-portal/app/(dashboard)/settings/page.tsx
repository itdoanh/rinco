"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LoadingSkeleton, ErrorBoundary } from "@rinco/ui";
import {
  Settings,
  Globe,
  Key,
  Bell,
  Database,
  Shield,
  Palette,
  Plus,
  Copy,
  Trash2,
  Save,
  RefreshCw,
  Zap,
  Mail,
  Server,
  Lock,
} from "lucide-react";
import { format } from "date-fns";
import { adminApi } from "@/lib/admin-api";

// Mock API Keys
const mockApiKeys = [
  { id: "key_001", name: "Production API Key", prefix: "rin_live_", lastUsed: "2026-09-17T10:30:00Z", createdAt: "2025-01-01T00:00:00Z", permissions: ["read", "write"] },
  { id: "key_002", name: "Development API Key", prefix: "rin_test_", lastUsed: "2026-09-17T09:15:00Z", createdAt: "2025-06-15T00:00:00Z", permissions: ["read", "write"] },
  { id: "key_003", name: "Analytics Read-Only", prefix: "rin_ro_", lastUsed: "2026-09-16T14:00:00Z", createdAt: "2025-09-01T00:00:00Z", permissions: ["read"] },
];

// Mock integrations
const mockIntegrations = [
  { id: "int_001", name: "Stripe", status: "connected", config: { apiVersion: "2023-10-16" } },
  { id: "int_002", name: "SendGrid", status: "connected", config: { fromEmail: "noreply@rinco.app" } },
  { id: "int_003", name: "Twilio", status: "connected", config: { region: "ap-southeast-1" } },
  { id: "int_004", name: "Sentry", status: "connected", config: { projectSlug: "rinco-prod" } },
  { id: "int_005", name: "Datadog", status: "disconnected", config: {} },
];

function SettingsInner() {
  const [activeTab, setActiveTab] = useState("general");
  const { data, isLoading } = useQuery({
    queryKey: ["platform-settings"],
    queryFn: async () => {
      const r = await adminApi.getPlatformSettings();
      return r.data ?? {};
    },
    staleTime: 60_000,
  });

  const platformName: string = (data?.platformName as string | undefined) ?? "RINCO Platform";

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <Settings className="w-7 h-7 text-primary" />
          Settings
        </h1>
        <p className="text-muted-foreground mt-1">
          System configuration, API keys, and integrations
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid grid-cols-5 w-auto">
          <TabsTrigger value="general" className="gap-1.5">
            <Settings className="w-4 h-4" />
            General
          </TabsTrigger>
          <TabsTrigger value="api-keys" className="gap-1.5">
            <Key className="w-4 h-4" />
            API Keys
          </TabsTrigger>
          <TabsTrigger value="integrations" className="gap-1.5">
            <Zap className="w-4 h-4" />
            Integrations
          </TabsTrigger>
          <TabsTrigger value="security" className="gap-1.5">
            <Shield className="w-4 h-4" />
            Security
          </TabsTrigger>
          <TabsTrigger value="notifications" className="gap-1.5">
            <Bell className="w-4 h-4" />
            Notifications
          </TabsTrigger>
        </TabsList>

        {/* General Settings */}
        <TabsContent value="general" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Globe className="w-4 h-4" />
                Platform Settings
              </CardTitle>
              <CardDescription>Configure global platform settings</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label htmlFor="platform-name">Platform Name</Label>
                  <Input id="platform-name" defaultValue={platformName} />
                </div>
                {isLoading && (
                  <div className="col-span-2">
                    <LoadingSkeleton className="h-9 w-full" />
                  </div>
                )}
                <div className="space-y-2">
                  <Label htmlFor="default-region">Default Region</Label>
                  <Select defaultValue="vn-sg">
                    <SelectTrigger id="default-region">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="vn-sg">Vietnam (Singapore)</SelectItem>
                      <SelectItem value="vn-hn">Vietnam (Hanoi)</SelectItem>
                      <SelectItem value="us-east">US East</SelectItem>
                      <SelectItem value="eu-west">EU West</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="text-sm font-medium flex items-center gap-2">
                  <Database className="w-4 h-4" />
                  Data Retention
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <Label>Log Retention (days)</Label>
                    <Input type="number" defaultValue={90} />
                  </div>
                  <div className="space-y-2">
                    <Label>Recording Retention (days)</Label>
                    <Input type="number" defaultValue={30} />
                  </div>
                  <div className="space-y-2">
                    <Label>Audit Log Retention (days)</Label>
                    <Input type="number" defaultValue={365} />
                  </div>
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="text-sm font-medium flex items-center gap-2">
                  <Server className="w-4 h-4" />
                  Infrastructure
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Cluster Mode</Label>
                    <Select defaultValue="hybrid">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="shared">Shared Cluster</SelectItem>
                        <SelectItem value="isolated">Isolated VPS</SelectItem>
                        <SelectItem value="hybrid">Hybrid</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Auto-scaling</Label>
                    <Select defaultValue="enabled">
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="enabled">Enabled</SelectItem>
                        <SelectItem value="disabled">Disabled</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>

              <div className="flex justify-end">
                <Button className="gap-2">
                  <Save className="w-4 h-4" />
                  Save Changes
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Branding */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Palette className="w-4 h-4" />
                Admin Branding
              </CardTitle>
              <CardDescription>Customize the admin portal appearance</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label>Logo URL</Label>
                  <Input placeholder="https://example.com/logo.png" />
                </div>
                <div className="space-y-2">
                  <Label>Favicon URL</Label>
                  <Input placeholder="https://example.com/favicon.ico" />
                </div>
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium">Dark Mode Default</div>
                  <div className="text-xs text-muted-foreground">Enable dark mode for all admin users</div>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* API Keys */}
        <TabsContent value="api-keys" className="space-y-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Key className="w-4 h-4" />
                  API Keys
                </CardTitle>
                <CardDescription>Manage API keys for external integrations</CardDescription>
              </div>
              <Button className="gap-2">
                <Plus className="w-4 h-4" />
                Create New Key
              </Button>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {mockApiKeys.map((key) => (
                  <div key={key.id} className="border rounded-lg p-4 hover:bg-slate-50/50 transition">
                    <div className="flex items-start justify-between gap-4 flex-wrap">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                          <h3 className="font-semibold">{key.name}</h3>
                          <Badge variant="outline">{key.prefix}****</Badge>
                        </div>
                        <div className="flex items-center gap-4 text-xs text-muted-foreground">
                          <span>Created: {format(new Date(key.createdAt), "yyyy-MM-dd")}</span>
                          <span>Last used: {format(new Date(key.lastUsed), "yyyy-MM-dd HH:mm")}</span>
                        </div>
                        <div className="flex items-center gap-2 mt-2">
                          {key.permissions.map((perm) => (
                            <Badge key={perm} variant="secondary" className="text-xs capitalize">
                              {perm}
                            </Badge>
                          ))}
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" className="gap-1.5">
                          <Copy className="w-3 h-3" />
                          Copy
                        </Button>
                        <Button variant="outline" size="sm" className="gap-1.5">
                          <RefreshCw className="w-3 h-3" />
                          Rotate
                        </Button>
                        <Button variant="outline" size="sm" className="gap-1.5 text-red-600 hover:text-red-700">
                          <Trash2 className="w-3 h-3" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Rate Limits */}
          <Card>
            <CardHeader>
              <CardTitle>Rate Limits</CardTitle>
              <CardDescription>Configure API rate limits per key</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label>Requests per minute</Label>
                  <Input type="number" defaultValue={1000} />
                </div>
                <div className="space-y-2">
                  <Label>Requests per hour</Label>
                  <Input type="number" defaultValue={50000} />
                </div>
                <div className="space-y-2">
                  <Label>Burst limit</Label>
                  <Input type="number" defaultValue={100} />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Integrations */}
        <TabsContent value="integrations" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {mockIntegrations.map((integration) => (
              <Card key={integration.id} className="hover:border-primary/50 transition-colors">
                <CardContent className="p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                        integration.status === "connected" ? "bg-emerald-100 text-emerald-700" : "bg-gray-100 text-gray-500"
                      }`}>
                        <Zap className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-semibold">{integration.name}</h3>
                        <Badge variant={integration.status === "connected" ? "default" : "secondary"} className="text-xs mt-1 capitalize">
                          {integration.status}
                        </Badge>
                      </div>
                    </div>
                  </div>
                  <div className="text-xs text-muted-foreground space-y-1">
                    {Object.entries(integration.config).map(([k, v]) => (
                      <div key={k}>
                        <span className="capitalize">{k.replace(/([A-Z])/g, " $1")}: </span>
                        <span className="font-mono">{String(v)}</span>
                      </div>
                    ))}
                  </div>
                  <div className="flex gap-2 mt-4">
                    <Button variant="outline" size="sm" className="flex-1">
                      Configure
                    </Button>
                    <Button variant={integration.status === "connected" ? "outline" : "default"} size="sm" className="flex-1">
                      {integration.status === "connected" ? "Disconnect" : "Connect"}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}

            {/* Add New Integration */}
            <Card className="border-dashed cursor-pointer hover:border-primary/50 transition-colors">
              <CardContent className="p-4 flex flex-col items-center justify-center min-h-[160px] text-muted-foreground">
                <Plus className="w-8 h-8 mb-2" />
                <div className="text-sm font-medium">Add Integration</div>
                <div className="text-xs">Connect a new service</div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Security */}
        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="w-4 h-4" />
                Security Settings
              </CardTitle>
              <CardDescription>Configure authentication and security policies</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium flex items-center gap-2">
                      <Lock className="w-4 h-4" />
                      Two-Factor Authentication
                    </div>
                    <div className="text-xs text-muted-foreground">Require 2FA for all admin users</div>
                  </div>
                  <Switch defaultChecked />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium flex items-center gap-2">
                      <Lock className="w-4 h-4" />
                      YubiKey Required
                    </div>
                    <div className="text-xs text-muted-foreground">Require hardware key for sensitive actions</div>
                  </div>
                  <Switch defaultChecked />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium flex items-center gap-2">
                      <Lock className="w-4 h-4" />
                      Session Timeout
                    </div>
                    <div className="text-xs text-muted-foreground">Auto-logout after inactivity</div>
                  </div>
                  <Select defaultValue="8h">
                    <SelectTrigger className="w-32">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1h">1 hour</SelectItem>
                      <SelectItem value="4h">4 hours</SelectItem>
                      <SelectItem value="8h">8 hours</SelectItem>
                      <SelectItem value="24h">24 hours</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <Separator />

                <div className="space-y-2">
                  <div className="text-sm font-medium flex items-center gap-2">
                    <Globe className="w-4 h-4" />
                    Allowed IP Ranges
                  </div>
                  <div className="text-xs text-muted-foreground mb-2">Restrict admin access to specific IPs</div>
                  <div className="space-y-2">
                    <Input placeholder="10.0.0.0/8" defaultValue="10.0.0.0/8" />
                    <Input placeholder="192.168.0.0/16" defaultValue="192.168.0.0/16" />
                  </div>
                </div>
              </div>

              <div className="flex justify-end">
                <Button className="gap-2">
                  <Save className="w-4 h-4" />
                  Save Security Settings
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Password Policy */}
          <Card>
            <CardHeader>
              <CardTitle>Password Policy</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Minimum Length</Label>
                  <Input type="number" defaultValue={12} />
                </div>
                <div className="space-y-2">
                  <Label>Password Expiry (days)</Label>
                  <Input type="number" defaultValue={90} />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Complexity Requirements</Label>
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Switch defaultChecked />
                    <span className="text-sm">Uppercase letters</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch defaultChecked />
                    <span className="text-sm">Lowercase letters</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch defaultChecked />
                    <span className="text-sm">Numbers</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch defaultChecked />
                    <span className="text-sm">Special characters</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Notifications Settings */}
        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bell className="w-4 h-4" />
                Notification Preferences
              </CardTitle>
              <CardDescription>Configure how and when you receive notifications</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h3 className="text-sm font-medium">Alert Channels</h3>
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Mail className="w-4 h-4 text-muted-foreground" />
                      <div>
                        <div className="text-sm font-medium">Email</div>
                        <div className="text-xs text-muted-foreground">owner@rinco.app</div>
                      </div>
                    </div>
                    <Switch defaultChecked />
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Zap className="w-4 h-4 text-muted-foreground" />
                      <div className="text-sm font-medium">Telegram</div>
                    </div>
                    <Switch defaultChecked />
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Bell className="w-4 h-4 text-muted-foreground" />
                      <div className="text-sm font-medium">In-App</div>
                    </div>
                    <Switch defaultChecked />
                  </div>
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="text-sm font-medium">Alert Types</h3>
                <div className="space-y-3">
                  {[
                    { name: "Critical Alerts", desc: "Service down, security breach, quota exceeded" },
                    { name: "Warning Alerts", desc: "High CPU, memory, error rate spikes" },
                    { name: "System Notifications", desc: "Deployments, maintenance, feature flags" },
                    { name: "Quorum Requests", desc: "Multi-party authorization approvals" },
                  ].map((alert) => (
                    <div key={alert.name} className="flex items-center justify-between">
                      <div>
                        <div className="text-sm font-medium">{alert.name}</div>
                        <div className="text-xs text-muted-foreground">{alert.desc}</div>
                      </div>
                      <Switch defaultChecked />
                    </div>
                  ))}
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <h3 className="text-sm font-medium">Quiet Hours</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Start Time</Label>
                    <Input type="time" defaultValue="22:00" />
                  </div>
                  <div className="space-y-2">
                    <Label>End Time</Label>
                    <Input type="time" defaultValue="07:00" />
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-sm font-medium">Weekend quiet hours</div>
                    <div className="text-xs text-muted-foreground">Silence non-critical alerts on weekends</div>
                  </div>
                  <Switch />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default function SettingsPage() {
  return (
    <ErrorBoundary>
      <SettingsInner />
    </ErrorBoundary>
  );
}
