"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Flag, Plus, Search, Beaker, Rocket, Star, Archive } from "lucide-react";
import { mockFeatureFlags, type MockFeatureFlag } from "@/lib/mock-data";

const categoryMeta: Record<MockFeatureFlag["category"], { color: string; icon: typeof Beaker }> = {
  core: { color: "bg-emerald-100 text-emerald-700", icon: Rocket },
  experimental: { color: "bg-purple-100 text-purple-700", icon: Beaker },
  beta: { color: "bg-amber-100 text-amber-700", icon: Star },
  deprecated: { color: "bg-gray-100 text-gray-700", icon: Archive },
};

const VALID_CATEGORIES: MockFeatureFlag["category"][] = ["core", "experimental", "beta", "deprecated"];

export default function FeatureFlagsPage() {
  const [flags, setFlags] = useState<MockFeatureFlag[]>(mockFeatureFlags);
  const [search, setSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<{
    key: string;
    description: string;
    category: MockFeatureFlag["category"];
    rolloutPercentage: number;
    enabled: boolean;
  }>({
    key: "",
    description: "",
    category: "experimental",
    rolloutPercentage: 0,
    enabled: false,
  });

  const filtered = flags.filter(
    (f) =>
      (categoryFilter === "all" || f.category === categoryFilter) &&
      (f.key.toLowerCase().includes(search.toLowerCase()) ||
        f.description.toLowerCase().includes(search.toLowerCase())),
  );

  function toggle(key: string) {
    setFlags((prev) =>
      prev.map((f) => (f.key === key ? { ...f, enabled: !f.enabled } : f)),
    );
  }

  function updateRollout(key: string, value: number) {
    setFlags((prev) =>
      prev.map((f) => (f.key === key ? { ...f, rolloutPercentage: value } : f)),
    );
  }

  function submit() {
    const key = draft.key.trim();
    if (!key) return;
    setFlags((prev) => [
      ...prev,
      {
        key,
        description: draft.description.trim() || "(no description)",
        category: draft.category,
        enabled: draft.enabled,
        rolloutPercentage: draft.enabled ? draft.rolloutPercentage : 0,
        updatedAt: new Date().toISOString(),
        updatedBy: "owner@rinco.app",
        tenantOverrides: 0,
      },
    ]);
    setDraft({ key: "", description: "", category: "experimental", rolloutPercentage: 0, enabled: false });
    setOpen(false);
  }

  const counts = {
    total: flags.length,
    enabled: flags.filter((f) => f.enabled).length,
    experimental: flags.filter((f) => f.category === "experimental").length,
    core: flags.filter((f) => f.category === "core").length,
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Flag className="w-7 h-7 text-primary" />
            Feature Flags
          </h1>
          <p className="text-muted-foreground mt-1">
            {counts.enabled}/{counts.total} flags enabled · {counts.experimental} experimental · {counts.core} core
          </p>
        </div>
        <div className="flex gap-2">
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Flag
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>New Feature Flag</DialogTitle>
                <DialogDescription>
                  Flags are checked at runtime by the application services.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="flag-key">Key</Label>
                  <Input
                    id="flag-key"
                    value={draft.key}
                    onChange={(e) => setDraft({ ...draft, key: e.target.value })}
                    placeholder="experiment.cool_feature"
                  />
                </div>
                <div>
                  <Label htmlFor="flag-desc">Description</Label>
                  <Input
                    id="flag-desc"
                    value={draft.description}
                    onChange={(e) => setDraft({ ...draft, description: e.target.value })}
                    placeholder="What this flag does"
                  />
                </div>
                <div>
                  <Label htmlFor="flag-cat">Category</Label>
                  <select
                    id="flag-cat"
                    className="w-full border-2 border-slate-200 bg-white rounded-md px-3 py-2 mt-1 text-sm"
                    value={draft.category}
                    onChange={(e) =>
                      setDraft({ ...draft, category: e.target.value as MockFeatureFlag["category"] })
                    }
                  >
                    {VALID_CATEGORIES.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <Label htmlFor="flag-rollout">Rollout %</Label>
                  <Input
                    id="flag-rollout"
                    type="number"
                    min={0}
                    max={100}
                    value={draft.rolloutPercentage}
                    onChange={(e) =>
                      setDraft({ ...draft, rolloutPercentage: Number(e.target.value) })
                    }
                  />
                </div>
                <label className="flex items-center gap-2 text-sm">
                  <Switch
                    checked={draft.enabled}
                    onCheckedChange={(v) => setDraft({ ...draft, enabled: v })}
                  />
                  Enabled
                </label>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={submit} disabled={!draft.key.trim()}>
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <Card>
        <CardHeader className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search flags by key or description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <div className="flex gap-1.5 flex-wrap">
            {(["all", "core", "experimental", "beta", "deprecated"] as const).map((c) => (
              <Button
                key={c}
                size="sm"
                variant={categoryFilter === c ? "default" : "outline"}
                onClick={() => setCategoryFilter(c)}
                className="capitalize"
              >
                {c}
              </Button>
            ))}
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-slate-50 text-left text-xs uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-3">Enabled</th>
                  <th className="py-3 px-3">Key</th>
                  <th className="py-3 px-3">Description</th>
                  <th className="py-3 px-3">Category</th>
                  <th className="py-3 px-3">Rollout</th>
                  <th className="py-3 px-3">Overrides</th>
                  <th className="py-3 px-3">Last Modified</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((f) => {
                  const meta = categoryMeta[f.category];
                  const Icon = meta.icon;
                  return (
                    <tr key={f.key} className="border-b hover:bg-slate-50/60">
                      <td className="py-3 px-3">
                        <Switch checked={f.enabled} onCheckedChange={() => toggle(f.key)} />
                      </td>
                      <td className="py-3 px-3">
                        <div className="font-mono text-xs font-semibold">{f.key}</div>
                      </td>
                      <td className="py-3 px-3 max-w-xs">
                        <div className="text-sm">{f.description}</div>
                      </td>
                      <td className="py-3 px-3">
                        <Badge className={`${meta.color} gap-1`}>
                          <Icon className="w-3 h-3" />
                          {f.category}
                        </Badge>
                      </td>
                      <td className="py-3 px-3 w-44">
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-slate-200 rounded-full h-1.5">
                            <div
                              className={`h-1.5 rounded-full transition-all ${f.enabled ? "bg-emerald-500" : "bg-slate-300"}`}
                              style={{ width: `${f.rolloutPercentage}%` }}
                            />
                          </div>
                          <Input
                            type="number"
                            min={0}
                            max={100}
                            value={f.rolloutPercentage}
                            onChange={(e) => updateRollout(f.key, Number(e.target.value))}
                            className="h-7 w-16 text-xs"
                            disabled={!f.enabled}
                          />
                        </div>
                      </td>
                      <td className="py-3 px-3">
                        <Badge variant="secondary" className="text-xs">
                          {f.tenantOverrides} tenants
                        </Badge>
                      </td>
                      <td className="py-3 px-3 text-xs text-muted-foreground">
                        <div>{new Date(f.updatedAt).toLocaleDateString()}</div>
                        <div className="text-[10px]">{f.updatedBy}</div>
                      </td>
                    </tr>
                  );
                })}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={7} className="text-center py-12 text-muted-foreground">
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
  );
}
