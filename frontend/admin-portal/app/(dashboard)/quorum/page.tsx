"use client";

import { useState } from "react";
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
import { Shield, Check, Clock, AlertTriangle, Plus } from "lucide-react";

interface QuorumRequest {
  id: string
  action: string
  reason: string
  initiator: string
  requiredSigs: number
  collectedSigs: string[]
  status: "PENDING" | "APPROVED" | "REJECTED" | "EXPIRED"
  expiresAt: string
  createdAt: string
}

const sampleQuorums: QuorumRequest[] = [
  {
    id: "qr-1",
    action: "tenant.delete",
    reason: "GDPR Right to be Forgotten request from customer (verified)",
    initiator: "security@rinco.app",
    requiredSigs: 2,
    collectedSigs: ["security@rinco.app", "owner@rinco.app"],
    status: "APPROVED",
    expiresAt: new Date(Date.now() - 3600_000).toISOString(),
    createdAt: new Date(Date.now() - 7200_000).toISOString(),
  },
  {
    id: "qr-2",
    action: "gateway.global_config",
    reason: "Emergency rate limit increase for upcoming marketing campaign",
    initiator: "ops@rinco.app",
    requiredSigs: 2,
    collectedSigs: ["ops@rinco.app"],
    status: "PENDING",
    expiresAt: new Date(Date.now() + 1800_000).toISOString(),
    createdAt: new Date(Date.now() - 600_000).toISOString(),
  },
  {
    id: "qr-3",
    action: "tenant.lock",
    reason: "Suspicious billing activity detected, locking for investigation",
    initiator: "finance@rinco.app",
    requiredSigs: 2,
    collectedSigs: [],
    status: "PENDING",
    expiresAt: new Date(Date.now() + 3000_000).toISOString(),
    createdAt: new Date(Date.now() - 120_000).toISOString(),
  },
]

const statusMeta = {
  PENDING: { icon: Clock, color: "text-amber-600", bg: "bg-amber-50", label: "PENDING" },
  APPROVED: { icon: Check, color: "text-emerald-600", bg: "bg-emerald-50", label: "APPROVED" },
  REJECTED: { icon: AlertTriangle, color: "text-red-600", bg: "bg-red-50", label: "REJECTED" },
  EXPIRED: { icon: Clock, color: "text-gray-500", bg: "bg-gray-50", label: "EXPIRED" },
}

export default function QuorumPage() {
  const [quorums, setQuorums] = useState(sampleQuorums)
  const [open, setOpen] = useState(false)
  const [newAction, setNewAction] = useState("")
  const [newReason, setNewReason] = useState("")

  function sign(id: string) {
    setQuorums(prev =>
      prev.map(q => {
        if (q.id !== id) return q
        if (q.status !== "PENDING") return q
        const updated = {
          ...q,
          collectedSigs: [...q.collectedSigs, "owner@rinco.app"],
        }
        if (updated.collectedSigs.length >= updated.requiredSigs) {
          updated.status = "APPROVED" as const
        }
        return updated
      })
    )
  }

  function createQuorum() {
    if (!newAction || !newReason) return
    const newQ: QuorumRequest = {
      id: "qr-" + Date.now(),
      action: newAction,
      reason: newReason,
      initiator: "owner@rinco.app",
      requiredSigs: 2,
      collectedSigs: ["owner@rinco.app"],
      status: "PENDING",
      expiresAt: new Date(Date.now() + 300_000).toISOString(),
      createdAt: new Date().toISOString(),
    }
    setQuorums(prev => [newQ, ...prev])
    setNewAction("")
    setNewReason("")
    setOpen(false)
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="w-6 h-6" />
            Multi-Party Authorization
          </h1>
          <p className="text-gray-500">
            2-of-3 YubiKey signing required for sensitive operations
          </p>
        </div>
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
                Select action & provide detailed reason. Will require 2 more admins to sign.
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
                  <option value="tenant.delete">Delete Tenant</option>
                  <option value="tenant.lock">Lock Tenant</option>
                  <option value="tenant.migrate">Migrate Tenant</option>
                  <option value="gateway.global_config">Gateway Global Config</option>
                  <option value="billing.refund">Issue Refund</option>
                  <option value="dns.rotate">Rotate DNS Keys</option>
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

      <Card>
        <CardHeader>
          <CardTitle>Active &amp; Recent Quorum Requests</CardTitle>
          <CardDescription>
            Each request expires in 5 minutes after creation. Required signatures: 2 of 3 OWNER / SRE_ADMIN / SECURITY_ADMIN.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {quorums.map(q => {
            const meta = statusMeta[q.status]
            const Icon = meta.icon
            const remaining = Math.round(
              (new Date(q.expiresAt).getTime() - Date.now()) / 1000
            )

            return (
              <div
                key={q.id}
                className={`border rounded-lg p-4 ${meta.bg} ${q.status === "APPROVED" ? "opacity-70" : ""}`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-2">
                      <Icon className={`w-5 h-5 ${meta.color}`} />
                      <h3 className="font-semibold">{q.action}</h3>
                      <Badge variant="outline" className={meta.color}>{meta.label}</Badge>
                      {q.status === "PENDING" && remaining > 0 && (
                        <Badge variant="outline" className="text-xs">
                          <Clock className="w-3 h-3 mr-1" />
                          {Math.floor(remaining / 60)}:{(remaining % 60).toString().padStart(2, "0")}
                        </Badge>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mb-2">{q.reason}</p>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground">
                      <span>Initiator: <span className="font-mono">{q.initiator}</span></span>
                      <span>Signatures: {q.collectedSigs.length}/{q.requiredSigs}</span>
                      <span>Created: {new Date(q.createdAt).toLocaleString()}</span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1">
                      {q.collectedSigs.map((sig, idx) => (
                        <Badge key={idx} variant="outline" className="text-xs font-mono">
                          <Check className="w-3 h-3 mr-1" />
                          {sig}
                        </Badge>
                      ))}
                    </div>
                  </div>
                  {q.status === "PENDING" && q.initiator !== "owner@rinco.app" && (
                    <Button
                      size="sm"
                      variant="default"
                      onClick={() => sign(q.id)}
                      disabled={q.collectedSigs.includes("owner@rinco.app")}
                    >
                      Sign with YubiKey
                    </Button>
                  )}
                </div>
              </div>
            )
          })}
          {quorums.length === 0 && (
            <div className="text-center py-12 text-muted-foreground">
              No quorum requests. Create one to require multi-party authorization.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
