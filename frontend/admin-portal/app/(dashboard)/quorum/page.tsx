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
import {
  Shield,
  Check,
  Clock,
  AlertTriangle,
  Plus,
  RotateCcw,
} from "lucide-react";
import {
  useQuorumStore,
  type QuorumRequest,
  type QuorumStatus,
} from "@/store/admin-stores";
import { formatRemaining, formatCountdown } from "@/lib/quorum-helpers";

const statusMeta: Record<
  QuorumStatus,
  { icon: typeof Shield; color: string; bg: string; label: string }
> = {
  PENDING: {
    icon: Clock,
    color: "text-amber-600",
    bg: "bg-amber-50",
    label: "PENDING",
  },
  APPROVED: {
    icon: Check,
    color: "text-emerald-600",
    bg: "bg-emerald-50",
    label: "APPROVED",
  },
  REJECTED: {
    icon: AlertTriangle,
    color: "text-red-600",
    bg: "bg-red-50",
    label: "REJECTED",
  },
  EXPIRED: {
    icon: Clock,
    color: "text-gray-500",
    bg: "bg-gray-50",
    label: "EXPIRED",
  },
};

const SUPPORTED_ACTIONS: { value: string; label: string }[] = [
  { value: "tenant.delete", label: "Delete Tenant" },
  { value: "tenant.lock", label: "Lock Tenant" },
  { value: "tenant.migrate", label: "Migrate Tenant" },
  { value: "gateway.global_config", label: "Gateway Global Config" },
  { value: "billing.refund", label: "Issue Refund" },
  { value: "dns.rotate", label: "Rotate DNS Keys" },
];

function formatRemainingLegacy(expiresAt: string, now: number): number {
  return Math.round((new Date(expiresAt).getTime() - now) / 1000);
}

function formatCountdownLegacy(seconds: number): string {
  if (seconds <= 0) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export default function QuorumPage() {
  const { requests, create, sign, reject, tickExpiry, reset } = useQuorumStore();

  // Touch hydration so SSR markup doesn't include persisted values.
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);
  const list: QuorumRequest[] = hydrated ? requests : [];

  // Tick expiry every second once hydrated so expired PENDING requests
  // get reclassified lazily without a backend.
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!hydrated) return;
    const id = window.setInterval(() => {
      tickExpiry();
      setNow(Date.now());
    }, 1000);
    return () => window.clearInterval(id);
  }, [hydrated, tickExpiry]);

  const sorted = useMemo(
    () =>
      [...list].sort(
        (a, b) =>
          new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
      ),
    [list],
  );

  const [open, setOpen] = useState(false);
  const [newAction, setNewAction] = useState("");
  const [newReason, setNewReason] = useState("");
  const [newSigs, setNewSigs] = useState(2);

  function createQuorum() {
    if (!newAction || newReason.length < 10) return;
    create({
      action: newAction,
      reason: newReason,
      initiator: "owner@rinco.app",
      requiredSigs: Math.max(1, Math.min(5, newSigs)),
      expiresAt: new Date(Date.now() + 5 * 60_000).toISOString(),
    });
    setNewAction("");
    setNewReason("");
    setNewSigs(2);
    setOpen(false);
  }

  const pending = sorted.filter((q) => q.status === "PENDING").length;
  const approved = sorted.filter((q) => q.status === "APPROVED").length;
  const rejected = sorted.filter((q) => q.status === "REJECTED").length;
  const expired = sorted.filter((q) => q.status === "EXPIRED").length;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="w-6 h-6" />
            Multi-Party Authorization
          </h1>
          <p className="text-gray-500">
            2-of-3 YubiKey signing required for sensitive operations. State
            persists in localStorage until the admin-gateway backend is
            wired up (docs/15-roadmap §2 Phase 2).
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
                New Quorum Request
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Quorum Request</DialogTitle>
                <DialogDescription>
                  Select an action & provide a detailed reason. Will require
                  the chosen number of additional admins to sign.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="action">Action</Label>
                  <select
                    id="action"
                    className="w-full border rounded-md px-3 py-2 mt-1"
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
                    onChange={(e) =>
                      setNewSigs(Number(e.target.value))
                    }
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={createQuorum}
                  disabled={!newAction || newReason.length < 10}
                >
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
            <div className="text-3xl font-bold text-amber-600">{pending}</div>
            <div className="text-xs text-muted-foreground uppercase">
              Pending
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-emerald-600">
              {approved}
            </div>
            <div className="text-xs text-muted-foreground uppercase">
              Approved
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-red-600">{rejected}</div>
            <div className="text-xs text-muted-foreground uppercase">
              Rejected
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 text-center">
            <div className="text-3xl font-bold text-gray-500">{expired}</div>
            <div className="text-xs text-muted-foreground uppercase">
              Expired
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Active &amp; Recent Quorum Requests</CardTitle>
          <CardDescription>
            Each request expires in 5 minutes after creation. Required
            signatures selected by the initiator (1-5).
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {sorted.map((q) => {
            const meta = statusMeta[q.status];
            const Icon = meta.icon;
            const remaining = formatRemaining(q.expiresAt, now);
            const isExpiredByTime =
              q.status === "PENDING" && remaining <= 0;

            return (
              <div
                key={q.id}
                className={`border rounded-lg p-4 ${meta.bg} ${q.status === "APPROVED" ? "opacity-70" : ""}`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-2 flex-wrap">
                      <Icon className={`w-5 h-5 ${meta.color}`} />
                      <h3 className="font-semibold">{q.action}</h3>
                      <Badge variant="outline" className={meta.color}>
                        {meta.label}
                      </Badge>
                      {q.status === "PENDING" && remaining > 0 && (
                        <Badge
                          variant="outline"
                          className="text-xs"
                          aria-label={`expires in ${formatCountdown(remaining)}`}
                        >
                          <Clock className="w-3 h-3 mr-1" />
                          {formatCountdown(remaining)}
                        </Badge>
                      )}
                      {isExpiredByTime && (
                        <Badge variant="outline" className="text-xs">
                          expiring...
                        </Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mb-2">
                      {q.reason}
                    </p>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground flex-wrap">
                      <span>
                        Initiator:{" "}
                        <span className="font-mono">{q.initiator}</span>
                      </span>
                      <span>
                        Signatures: {q.collectedSigs.length}/{q.requiredSigs}
                      </span>
                      <span>
                        Created: {new Date(q.createdAt).toLocaleString()}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {q.collectedSigs.map((sig, idx) => (
                        <Badge
                          key={idx}
                          variant="outline"
                          className="text-xs font-mono"
                        >
                          <Check className="w-3 h-3 mr-1" />
                          {sig}
                        </Badge>
                      ))}
                    </div>
                  </div>
                  {q.status === "PENDING" && (
                    <div className="flex flex-col gap-1">
                      <Button
                        size="sm"
                        variant="default"
                        onClick={() => sign(q.id, "owner@rinco.app")}
                        disabled={q.collectedSigs.includes(
                          "owner@rinco.app",
                        )}
                        title="Sign with YubiKey"
                      >
                        Sign
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => reject(q.id, "owner@rinco.app")}
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
              No quorum requests. Create one to require multi-party
              authorization.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
