"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Bell, Plus, Edit2, Trash2, Send, Code, Variable } from "lucide-react";
import { mockNotificationTemplates, type MockNotificationTemplate } from "@/lib/mock-data";

const categoryColor: Record<MockNotificationTemplate["category"], string> = {
  critical: "bg-red-100 text-red-700",
  warning: "bg-amber-100 text-amber-700",
  info: "bg-blue-100 text-blue-700",
  marketing: "bg-purple-100 text-purple-700",
};

const CATEGORIES: MockNotificationTemplate["category"][] = ["critical", "warning", "info", "marketing"];

const CHANNELS: MockNotificationTemplate["channel"][] = ["in_app", "email", "telegram", "push", "sms"];

function extractVariables(s: string): string[] {
  const re = /\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}/g;
  const found = new Set<string>();
  let m: RegExpExecArray | null;
  while ((m = re.exec(s)) !== null) {
    found.add(m[1]);
  }
  return Array.from(found);
}

interface DraftTemplate {
  code: string;
  category: MockNotificationTemplate["category"];
  channel: MockNotificationTemplate["channel"];
  subject: string;
  bodyTemplate: string;
}

const EMPTY_DRAFT: DraftTemplate = {
  code: "",
  category: "info",
  channel: "email",
  subject: "",
  bodyTemplate: "",
};

export default function NotificationTemplatesPage() {
  const [templates, setTemplates] = useState<MockNotificationTemplate[]>(mockNotificationTemplates);
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<DraftTemplate>(EMPTY_DRAFT);
  const [search, setSearch] = useState("");

  const filtered = templates.filter(
    (t) =>
      !search ||
      t.code.toLowerCase().includes(search.toLowerCase()) ||
      t.subject.toLowerCase().includes(search.toLowerCase()),
  );

  function startCreate() {
    setEditingId(null);
    setDraft(EMPTY_DRAFT);
    setOpen(true);
  }

  function startEdit(t: MockNotificationTemplate) {
    setEditingId(t.id);
    setDraft({
      code: t.code,
      category: t.category,
      channel: t.channel,
      subject: t.subject,
      bodyTemplate: t.bodyTemplate,
    });
    setOpen(true);
  }

  function submit() {
    if (!draft.code.trim() || !draft.subject.trim()) return;
    const subjectVars = extractVariables(draft.subject);
    const bodyVars = extractVariables(draft.bodyTemplate);
    const variables = Array.from(new Set([...subjectVars, ...bodyVars])).map((name) => ({
      name,
      type: "string" as const,
      required: true,
    }));

    if (editingId) {
      setTemplates((prev) =>
        prev.map((t) =>
          t.id === editingId
            ? { ...t, ...draft, variables, updatedAt: new Date().toISOString() }
            : t,
        ),
      );
    } else {
      const newTemplate: MockNotificationTemplate = {
        id: `tpl_${Date.now().toString(36)}`,
        code: draft.code,
        category: draft.category,
        channel: draft.channel,
        subject: draft.subject,
        bodyTemplate: draft.bodyTemplate,
        variables,
        updatedAt: new Date().toISOString(),
      };
      setTemplates((prev) => [...prev, newTemplate]);
    }
    setOpen(false);
  }

  function remove(id: string) {
    setTemplates((prev) => prev.filter((t) => t.id !== id));
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Bell className="w-7 h-7 text-primary" />
            Notification Templates
          </h1>
          <p className="text-muted-foreground mt-1">
            {templates.length} templates · supports <code className="font-mono text-xs">{"{{variable}}"}</code> placeholders
          </p>
        </div>
        <Button onClick={startCreate}>
          <Plus className="w-4 h-4 mr-2" />
          New Template
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Input
          placeholder="Search by code or subject..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="max-w-sm"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {filtered.map((t) => (
          <Card key={t.id} className="hover:border-primary/50 transition-colors">
            <CardHeader>
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <Code className="w-4 h-4 text-slate-400" />
                    <CardTitle className="text-sm font-mono">{t.code}</CardTitle>
                  </div>
                  <div className="flex items-center gap-2 flex-wrap mt-1">
                    <Badge className={categoryColor[t.category]}>{t.category}</Badge>
                    <Badge variant="outline">{t.channel.toUpperCase()}</Badge>
                  </div>
                  <CardDescription className="mt-2">Subject: {t.subject}</CardDescription>
                </div>
                <div className="flex gap-1 shrink-0">
                  <Button variant="ghost" size="sm" onClick={() => startEdit(t)} aria-label="Edit">
                    <Edit2 className="w-4 h-4" />
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => alert(`Test sent for template ${t.code}`)} aria-label="Send test">
                    <Send className="w-4 h-4" />
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => remove(t.id)} aria-label="Delete">
                    <Trash2 className="w-4 h-4 text-red-500" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-slate-700 bg-slate-50 p-3 rounded-md font-mono whitespace-pre-wrap border border-slate-100">
                {t.bodyTemplate}
              </div>
              <div className="mt-3 flex flex-wrap items-center gap-1.5">
                <Variable className="w-3 h-3 text-slate-400" />
                <span className="text-xs text-muted-foreground">Variables:</span>
                {t.variables.map((v, idx) => (
                  <span key={idx} className="text-xs font-mono bg-blue-50 text-blue-700 px-2 py-0.5 rounded">
                    {`{{${v.name}}}`}
                  </span>
                ))}
                {t.variables.length === 0 && (
                  <span className="text-xs text-muted-foreground italic">(none)</span>
                )}
              </div>
              <div className="mt-2 text-xs text-muted-foreground">
                Updated {new Date(t.updatedAt).toLocaleString()}
              </div>
            </CardContent>
          </Card>
        ))}
        {filtered.length === 0 && (
          <div className="col-span-2 text-center py-12 text-muted-foreground">
            No notification templates match your search.
          </div>
        )}
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {editingId ? "Edit Template" : "New Notification Template"}
            </DialogTitle>
            <DialogDescription>
              Use <code className="font-mono">{`{{variable_name}}`}</code> placeholders. Variables are auto-extracted on save.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="tpl-code">Code</Label>
              <Input
                id="tpl-code"
                value={draft.code}
                onChange={(e) => setDraft({ ...draft, code: e.target.value })}
                placeholder="e.g. billing.invoice_failed"
                disabled={editingId !== null}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="tpl-cat">Category</Label>
                <select
                  id="tpl-cat"
                  className="w-full border-2 border-slate-200 bg-white rounded-md px-3 py-2 mt-1 text-sm capitalize"
                  value={draft.category}
                  onChange={(e) =>
                    setDraft({ ...draft, category: e.target.value as MockNotificationTemplate["category"] })
                  }
                >
                  {CATEGORIES.map((c) => (
                    <option key={c} value={c} className="capitalize">{c}</option>
                  ))}
                </select>
              </div>
              <div>
                <Label htmlFor="tpl-channel">Channel</Label>
                <select
                  id="tpl-channel"
                  className="w-full border-2 border-slate-200 bg-white rounded-md px-3 py-2 mt-1 text-sm uppercase"
                  value={draft.channel}
                  onChange={(e) =>
                    setDraft({ ...draft, channel: e.target.value as MockNotificationTemplate["channel"] })
                  }
                >
                  {CHANNELS.map((c) => (
                    <option key={c} value={c}>{c.toUpperCase()}</option>
                  ))}
                </select>
              </div>
            </div>
            <div>
              <Label htmlFor="tpl-subject">Subject</Label>
              <Input
                id="tpl-subject"
                value={draft.subject}
                onChange={(e) => setDraft({ ...draft, subject: e.target.value })}
                placeholder="Email subject line"
              />
            </div>
            <div>
              <Label htmlFor="tpl-body">Body</Label>
              <Textarea
                id="tpl-body"
                value={draft.bodyTemplate}
                onChange={(e) => setDraft({ ...draft, bodyTemplate: e.target.value })}
                placeholder="Template body — supports {{variables}}"
                rows={6}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={submit} disabled={!draft.code.trim() || !draft.subject.trim()}>
              {editingId ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
