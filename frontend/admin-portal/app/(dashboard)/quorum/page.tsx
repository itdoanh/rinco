"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { LoadingSkeleton, EmptyState, ErrorBoundary } from "@rinco/ui";
import {
  Shield,
  Check,
  Clock,
  AlertTriangle,
  Plus,
  Timer,
} from "lucide-react";
import { adminApi, type QuorumRequest } from "@/lib/admin-api";

const statusMeta: Record<QuorumRequest["status"], {
  icon: typeof Shield;
  color: string;
  bg: string;
  label: string;
}> = {
  pending: { icon: Clock, color: "text-amber-700", bg: "bg-amber-50 border-amber-200", label: "PENDING" },
  approved: { icon: Check, color: "text-emerald-700", bg: "bg-emerald-50 border-emerald-200", label: "APPROVED" },
  rejected: { icon: AlertTriangle, color: "text-red-700", bg: "bg-red-50 border-red-200", label: "REJECTED" },
  expired: { icon: Clock, color: "text-gray-600", bg: "bg-gray-50 border-gray-200", label: "EXPIRED" },
};

const SUPPORTED_ACTIONS: { value: string; label: string }[] = [
  { value: "tenant.delete", label: "Delete Tenant" },
  { value: "tenant.lock", label: "Lock Tenant" },
  { value: "tenant.migrate", label: "Migrate Tenant" },
  { value: "gateway.global_config", label: "Gateway Global Config" },
  { value: "billing.refund", label: "Issue Refund" },
  { value: "dns.master_change", label: "Rotate DNS Master" },
];

function formatCountdown(seconds: number): string {
  if (seconds <= 0) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function QuorumInner() {
  const qc = useQueryClient();
  const [now, setNow] = useState(Date.now());
  const [open, setOpen] = useState(false);
  const [newAction, setNewAction] = useState("");
  const [newReason, setNewReason] = useState("");
  const [newSigs, setNewSigs] = useState(2);

  const { data, isLoading, error } = useQuery({
    queryKey: ["quorum"],
    queryFn: async () => {
      const r = await adminApi.getQuorumRequests();
      return r.data ?? [];
    },
    staleTime: 15_000,
  });

  const createMut = useMutation({
    mutationFn: (q: Partial<QuorumRequest>) => adminApi.createQuorumRequest(q),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["quorum"] }),
  });
  const signMut = useMutation({
    mutationFn: (id: string) => adminApi.signQuorumRequest(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["quorum"] }),
  });
  const rejectMut = useMutation({
    mutationFn: (id: string) => adminApi.rejectQuorumRequest(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["quorum"] }),
  });

  const requests: QuorumRequest[] = data ?? [];

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const sorted = useMemo(
    () =>
      [...requests].sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
      ),
    [requests],
  );

  function createQuorum() {
    if (!newAction || newReason.length < 10) return;
    createMut.mutate({
      action: newAction,
      reason: newReason,
      requiredSignatures: Math.max(1, Math.min(5, newSigs)),
    });
    setNewAction("");
    setNewReason("");
    setNewSigs(2);
    setOpen(false);
  }

  const pending = sorted.filter((q) => q.status === "pending").length;
  const approved = sorted.filter((q) => q.status === "approved").length;
  const rejected = sorted.filter((q) => q.status === "rejected").length;
  const expired = sorted.filter((q) => q.status === "expired").length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Shield className="w-7 h-7 text-primary" />
            Multi-Party Authorization
          </h1>
          <p className="text-muted-foreground mt-1">
            2-of-3 YubiKey signing required cho các thao tác nguy hiểm · tenant-service :8082
          </p>
        </div>
        <div className="flex gap-2">
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                New Quorum Request
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Quorum Request</DialogTitle>
                <DialogDescription>
                  Select an action & provide a detailed reason. Will require the chosen number of additional admins to sign.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="action">Action</Label>
                  <select
                    id="action"
                    className="w-full border-2 border-slate-200 bg-white rounded-md px-3 py-2 mt-1 text-sm"
                    value={newAction}
                    onChange={(e) => setNewAction(e.target.value)}
                  >
                    <option value="">-- Select action --</option>
                    {SUPPORTED_ACTIONS.map((a) => (
                      <option key={a.value} value={a.value}>
                        {a.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <Label htmlFor="reason">Reason (min 10 chars)</Label>
                  <Input
                    id="reason"
                    value={newReason}
                    onChange={(e) => setNewReason(e.target.value)}
                    placeholder="e.g., Customer GDPR request ticket #12345"
                  />
                </div>
                <div>
                  <Label htmlFor="reqsigs">Required signatures</Label>
                  <Input
                    id="reqsigs"
                    type="number"
                    min={1}
                    max={5}
                    value={newSigs}
                    onChange={(e) => setNewSigs(Number(e.target.value))}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={createQuorum}
                  disabled={
                    !newAction ||
                    newReason.length < 10 ||
                    createMut.isPending
                  }
                >
                  Create Request
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatTile value={pending} label="Pending" tone="amber" loading={isLoading} />
        <StatTile value={approved} label="Approved" tone="emerald" loading={isLoading} />
        <StatTile value={rejected} label="Rejected" tone="red" loading={isLoading} />
        <StatTile value={expired} label="Expired" tone="slate" loading={isLoading} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Active &amp; Recent Quorum Requests</CardTitle>
          <CardDescription>
            Each request expires in 5 minutes after creation.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <LoadingSkeleton key={i} className="h-24 w-full" />
              ))}
            </div>
          ) : error ? (
            <EmptyState
              variant="error"
              title="Không tải được quorum"
              description={(error as Error).message}
            />
          ) : sorted.length === 0 ? (
            <EmptyState
              variant="default"
              title="No quorum requests"
              description="Create one to require multi-party authorization."
            />
          ) : (
            sorted.map((q) => {
              const meta = statusMeta[q.status] ?? statusMeta.pending;
              const Icon = meta.icon;
              const remaining = Math.round(
                (new Date(q.expiresAt).getTime() - now) / 1000,
              );
              const isExpiring = q.status === "pending" && remaining > 0 && remaining < 60;
              const isExpired = q.status === "pending" && remaining <= 0;
              const alreadySigned = q.collectedSignatures.some(
                (s) => s.email === "owner@rinco.app",
              );

              return (
                <div
                  key={q.id}
                  className={`border rounded-xl p-4 transition ${meta.bg} ${
                    q.status === "approved" ? "opacity-80" : ""
                  }`}
                >
                  <div className="flex items-start justify-between gap-4 flex-wrap">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-2 flex-wrap">
                        <Icon className={`w-5 h-5 ${meta.color}`} />
                        <h3 className="font-semibold font-mono text-sm">{q.action}</h3>
                        <Badge variant="outline" className={meta.color}>
                          {meta.label}
                        </Badge>
                        {q.status === "pending" && remaining > 0 && (
                          <Badge
                            variant="outline"
                            className={`text-xs ${
                              isExpiring
                                ? "bg-red-50 text-red-700 border-red-300 animate-pulse"
                                : ""
                            }`}
                          >
                            <Timer className="w-3 h-3 mr-1" />
                            {formatCountdown(remaining)}
                          </Badge>
                        )}
                        {isExpired && (
                          <Badge variant="destructive" className="text-xs">
                            EXPIRED
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-slate-700 mb-2 italic">"{q.reason}"</p>
                      <div className="flex items-center gap-4 text-xs text-slate-600 flex-wrap">
                        <span>
                          Initiator:{" "}
                          <span className="font-mono font-semibold">
                            {q.initiatorEmail}
                          </span>
                        </span>
                        <span>
                          Signatures:{" "}
                          <span className="font-bold">
                            {q.collectedSignatures.length}/{q.requiredSignatures}
                          </span>
                        </span>
                        <span>
                          Created: {new Date(q.createdAt).toLocaleString()}
                        </span>
                      </div>
                      {q.collectedSignatures.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {q.collectedSignatures.map((sig, idx) => (
                            <Badge
                              key={idx}
                              variant="outline"
                              className="text-xs font-mono bg-white/50"
                            >
                              <Check className="w-3 h-3 mr-1 text-emerald-600" />
                              {sig.email}
                            </Badge>
                          ))}
                        </div>
                      )}
                    </div>
                    {q.status === "pending" && (
                      <div className="flex flex-col gap-1.5 shrink-0">
                        <Button
                          size="sm"
                          onClick={() => signMut.mutate(q.id)}
                          disabled={alreadySigned || signMut.isPending}
                          title="Sign with YubiKey"
                        >
                          <Check className="w-3 h-3 mr-1" /> Sign
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => rejectMut.mutate(q.id)}
                          disabled={rejectMut.isPending}
                        >
                          Reject
                        </Button>
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function StatTile({
  value,
  label,
  tone,
  loading,
}: {
  value: number;
  label: string;
  tone: "amber" | "emerald" | "red" | "slate";
  loading?: boolean;
}) {
  const tones = {
    amber: "text-amber-600",
    emerald: "text-emerald-600",
    red: "text-red-600",
    slate: "text-gray-500",
  };
  return (
    <Card>
      <CardContent className="p-4 text-center">
        {loading ? (
          <LoadingSkeleton className="h-8 w-12 mx-auto" />
        ) : (
          <div className={`text-3xl font-bold tabular-nums ${tones[tone]}`}>{value}</div>
        )}
        <div className="text-xs text-muted-foreground uppercase tracking-wider">{label}</div>
      </CardContent>
    </Card>
  );
}

export default function QuorumPage() {
  return (
    <ErrorBoundary>
      <QuorumInner />
    </ErrorBoundary>
  );
}
