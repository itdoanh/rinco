"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Flag, Plus, Search } from "lucide-react";

interface FeatureFlag {
  key: string
  description: string
  enabled: boolean
  rolloutPercentage: number
  category: "core" | "experimental" | "beta" | "deprecated"
  lastModified: string
}

const sampleFlags: FeatureFlag[] = [
  {
    key: "dynamic_model_engine",
    description: "Enable dynamic schema generation for tenant CRM models",
    enabled: true,
    rolloutPercentage: 100,
    category: "core",
    lastModified: new Date(Date.now() - 86400_000).toISOString(),
  },
  {
    key: "ai_lead_scoring_v2",
    description: "Use ML-based lead scoring v2 with pLTV prediction",
    enabled: true,
    rolloutPercentage: 75,
    category: "beta",
    lastModified: new Date(Date.now() - 3600_000).toISOString(),
  },
  {
    key: "e2ee_chat",
    description: "End-to-end encryption for chat messages (Signal Protocol)",
    enabled: true,
    rolloutPercentage: 50,
    category: "experimental",
    lastModified: new Date(Date.now() - 7200_000).toISOString(),
  },
  {
    key: "webrtc_av1_svc",
    description: "Enable AV1 SVC encoding for WebRTC SFU",
    enabled: false,
    rolloutPercentage: 0,
    category: "experimental",
    lastModified: new Date(Date.now() - 172800_000).toISOString(),
  },
  {
    key: "facebook_capi_v2",
    description: "Send events to FB Conversions API v2",
    enabled: true,
    rolloutPercentage: 100,
    category: "core",
    lastModified: new Date(Date.now() - 259200_000).toISOString(),
  },
  {
    key: "legacy_landing_v1",
    description: "Old jQuery-based landing page renderer (deprecated)",
    enabled: false,
    rolloutPercentage: 0,
    category: "deprecated",
    lastModified: new Date(Date.now() - 864000_000).toISOString(),
  },
]

const categoryColor = {
  core: "bg-emerald-100 text-emerald-700",
  experimental: "bg-purple-100 text-purple-700",
  beta: "bg-amber-100 text-amber-700",
  deprecated: "bg-gray-100 text-gray-700",
}

export default function FeatureFlagsPage() {
  const [flags, setFlags] = useState(sampleFlags)
  const [search, setSearch] = useState("")

  const filtered = flags.filter(f =>
    f.key.toLowerCase().includes(search.toLowerCase()) ||
    f.description.toLowerCase().includes(search.toLowerCase())
  )

  function toggle(key: string) {
    setFlags(prev =>
      prev.map(f => f.key === key ? { ...f, enabled: !f.enabled } : f)
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Flag className="w-6 h-6" />
            Feature Flags
          </h1>
          <p className="text-gray-500">
            Control feature availability per tenant and rollout percentage.
          </p>
        </div>
        <Button>
          <Plus className="w-4 h-4 mr-2" />
          New Flag
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search flags by key or description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground text-xs uppercase">
                  <th className="text-left py-3 px-3">Enabled</th>
                  <th className="text-left py-3 px-3">Key</th>
                  <th className="text-left py-3 px-3">Description</th>
                  <th className="text-left py-3 px-3">Category</th>
                  <th className="text-left py-3 px-3">Rollout</th>
                  <th className="text-left py-3 px-3">Last Modified</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map(f => (
                  <tr key={f.key} className="border-b hover:bg-muted/40">
                    <td className="py-3 px-3">
                      <Switch
                        checked={f.enabled}
                        onCheckedChange={() => toggle(f.key)}
                      />
                    </td>
                    <td className="py-3 px-3 font-mono text-xs">{f.key}</td>
                    <td className="py-3 px-3">{f.description}</td>
                    <td className="py-3 px-3">
                      <Badge className={categoryColor[f.category]}>{f.category}</Badge>
                    </td>
                    <td className="py-3 px-3">
                      {f.enabled ? `${f.rolloutPercentage}%` : "—"}
                    </td>
                    <td className="py-3 px-3 text-xs text-muted-foreground">
                      {new Date(f.lastModified).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={6} className="text-center py-12 text-muted-foreground">
                      No matching feature flags.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
