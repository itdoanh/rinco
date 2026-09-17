"use client";

import { useEffect, useMemo, useState } from "react";
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
import { Shield, Check, Clock, AlertTriangle, Plus, Timer } from "lucide-react";
import { mockQuorumRequests, type MockQuorumRequest } from "@/lib/mock-data";

const statusMeta: Record<MockQuorumRequest["status"], {
  icon: typeof Shield;
  color: string;
  bg: string;
  label: string;
}> = {
  pending: {
    icon: Clock,
    color: "text-amber-700",
    bg: "bg-amber-50 border-amber-200",
    label: "PENDING",
  },
  approved: {
    icon: Check,
    color: "text-emerald-700",
    bg: "bg-emerald-50 border-emerald-200",
    label: "APPROVED",
  },
  rejected: {
    icon: AlertTriangle,
    color: "text-red-700",
    bg: "bg-red-50 border-red-200",
    label: "REJECTED",
  },
  expired: {
    icon: Clock,
    color: "text-gray-600",
    bg: "bg-gray-50 border-gray-200",
    label: "EXPIRED",
  },
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

export default function QuorumPage() {
  const [requests, setRequests] = useState<MockQuorumRequest[]>(mockQuorumRequests);
  const [now, setNow] = useState(Date.now());
  const [open, setOpen] = useState(false);
  const [newAction, setNewAction] = useState("");
  const [newReason, setNewReason] = useState("");
  const [newSigs, setNewSigs] = useState(2);

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
    setRequests((prev) => [
      ...prev,
      {
        id: `qr_${Date.now().toString(36)}`,
        action: newAction,
        payload: {},
        initiatorEmail: "owner@rinco.app",
        reason: newReason,
        requiredSignatures: Math.max(1, Math.min(5, newSigs)),
        collectedSignatures: [
          { email: "owner@rinco.app", signedAt: new Date().toISOString() },
        ],
        status: "pending",
        expiresAt: new Date(Date.now() + 5 * 60_000).toISOString(),
        createdAt: new Date().toISOString(),
      },
    ]);
    setNewAction("");
    setNewReason("");
    setNewSigs(2);
    setOpen(false);
  }

  function sign(id: string) {
    setRequests((prev) =>
      prev.map((q) => {
        if (q.id !== id) return q;
        if (q.collectedSignatures.some((s) => s.email === "owner@rinco.app")) return q;
        const newSigs = [
          ...q.collectedSignatures,
          { email: "owner@rinco.app", signedAt: new Date().toISOString() },
        ];
        return {
          ...q,
          collectedSignatures: newSigs,
          status: newSigs.length >= q.requiredSignatures ? "approved" : "pending",
        };
      }),
    );
  }

  function reject(id: string) {
    setRequests((prev) =>
      prev.map((q) => (q.id === id ? { ...q, status: "rejected" } : q)),
    );
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
            2-of-3 YubiKey signing required cho các thao tác nguy hiểm
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
                <Button onClick={createQuorum} disabled={!newAction || newReason.length < 10}>
                  Create Request
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-amber-600 tabular-nums">{pending}</div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider">Pending</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-emerald-600 tabular-nums">{approved}</div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider">Approved</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-red-600 tabular-nums">{rejected}</div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider">Rejected</div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-gray-500 tabular-nums">{expired}</div>
            <div className="text-xs text-muted-foreground uppercase tracking-wider">Expired</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Active &amp; Recent Quorum Requests</CardTitle>
          <CardDescription>
            Each request expires in 5 minutes after creation. Required signatures selected by the initiator (1-5).
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {sorted.map((q) => {
            const meta = statusMeta[q.status];
            const Icon = meta.icon;
            const remaining = Math.round((new Date(q.expiresAt).getTime() - now) / 1000);
            const isExpiring = q.status === "pending" && remaining > 0 && remaining < 60;
            const isExpired = q.status === "pending" && remaining <= 0;

            return (
              <div
                key={q.id}
                className={`border rounded-xl p-4 transition ${meta.bg} ${q.status === "approved" ? "opacity-80" : ""}`}
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
                          className={`text-xs ${isExpiring ? "bg-red-50 text-red-700 border-red-300 animate-pulse" : ""}`}
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
                        Initiator: <span className="font-mono font-semibold">{q.initiatorEmail}</span>
                      </span>
                      <span>
                        Signatures: <span className="font-bold">{q.collectedSignatures.length}/{q.requiredSignatures}</span>
                      </span>
                      <span>Created: {new Date(q.createdAt).toLocaleString()}</span>
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
                        onClick={() => sign(q.id)}
                        disabled={q.collectedSignatures.some((s) => s.email === "owner@rinco.app")}
                        title="Sign with YubiKey"
                      >
                        <Check className="w-3 h-3 mr-1" /> Sign
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => reject(q.id)}
                      >
                        Reject
                      </Button>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
          {sorted.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              No quorum requests. Create one to require multi-party authorization.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
