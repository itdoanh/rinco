"use client";

import { useEffect, useState } from "react";
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
import { Flag, Plus, Search, RotateCcw } from "lucide-react";
import {
  useFeatureFlagsStore,
  type FeatureFlag,
  type FeatureFlagCategory,
} from "@/store/admin-stores";

const categoryColor: Record<FeatureFlagCategory, string> = {
  core: "bg-emerald-100 text-emerald-700",
  experimental: "bg-purple-100 text-purple-700",
  beta: "bg-amber-100 text-amber-700",
  deprecated: "bg-gray-100 text-gray-700",
};

const VALID_CATEGORIES: FeatureFlagCategory[] = [
  "core",
  "experimental",
  "beta",
  "deprecated",
];

export default function FeatureFlagsPage() {
  const { flags, toggle, updateRollout, add, remove, reset } =
    useFeatureFlagsStore();
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<{
    key: string;
    description: string;
    category: FeatureFlagCategory;
    rolloutPercentage: number;
    enabled: boolean;
  }>({
    key: "",
    description: "",
    category: "experimental",
    rolloutPercentage: 0,
    enabled: false,
  });

  // Touch hydration side-effect to avoid SSR mismatches.
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);
  const list: FeatureFlag[] = hydrated ? flags : [];

  const filtered = list.filter(
    (f) =>
      f.key.toLowerCase().includes(search.toLowerCase()) ||
      f.description.toLowerCase().includes(search.toLowerCase()),
  );

  function submit() {
    const key = draft.key.trim();
    if (!key) return;
    add({
      key,
      description: draft.description.trim() || "(no description)",
      category: draft.category,
      enabled: draft.enabled,
      rolloutPercentage: draft.enabled ? draft.rolloutPercentage : 0,
    });
    setDraft({
      key: "",
      description: "",
      category: "experimental",
      rolloutPercentage: 0,
      enabled: false,
    });
    setOpen(false);
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
            State persists in localStorage until the admin-gateway backend
            is wired up (docs/15-roadmap §2 Phase 2).
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={reset} title="Reset to defaults">
            <RotateCcw className="w-4 h-4" />
          </Button>
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
                    onChange={(e) =>
                      setDraft({ ...draft, key: e.target.value })
                    }
                    placeholder="experiment.cool_feature"
                  />
                </div>
                <div>
                  <Label htmlFor="flag-desc">Description</Label>
                  <Input
                    id="flag-desc"
                    value={draft.description}
                    onChange={(e) =>
                      setDraft({ ...draft, description: e.target.value })
                    }
                    placeholder="What this flag does"
                  />
                </div>
                <div>
                  <Label htmlFor="flag-cat">Category</Label>
                  <select
                    id="flag-cat"
                    className="w-full border rounded-md px-3 py-2 mt-1"
                    value={draft.category}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        category: e.target.value as FeatureFlagCategory,
                      })
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
                      setDraft({
                        ...draft,
                        rolloutPercentage: Number(e.target.value),
                      })
                    }
                  />
                </div>
                <label className="flex items-center gap-2 text-sm">
                  <Switch
                    checked={draft.enabled}
                    onCheckedChange={(v) =>
                      setDraft({ ...draft, enabled: v })
                    }
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
                  <th className="text-right py-3 px-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((f) => (
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
                      <Badge className={categoryColor[f.category]}>
                        {f.category}
                      </Badge>
                    </td>
                    <td className="py-3 px-3 w-32">
                      <Input
                        type="number"
                        min={0}
                        max={100}
                        value={f.rolloutPercentage}
                        onChange={(e) =>
                          updateRollout(f.key, Number(e.target.value))
                        }
                        className="h-7 w-20 text-xs"
                        disabled={!f.enabled}
                      />
                      <span className="text-xs text-muted-foreground ml-1">
                        %
                      </span>
                    </td>
                    <td className="py-3 px-3 text-xs text-muted-foreground">
                      {new Date(f.lastModified).toLocaleString()}
                    </td>
                    <td className="py-3 px-3 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => remove(f.key)}
                      >
                        Delete
                      </Button>
                    </td>
                  </tr>
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td
                      colSpan={7}
                      className="text-center py-12 text-muted-foreground"
                    >
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
