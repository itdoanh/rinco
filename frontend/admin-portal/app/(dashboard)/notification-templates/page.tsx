"use client";

import { useEffect, useState } from "react";
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
import { Bell, Plus, Edit2, Trash2, Send, RotateCcw } from "lucide-react";
import {
  useNotificationTemplatesStore,
  type NotificationTemplate,
  type NotificationCategory,
  type NotificationChannel,
} from "@/store/admin-stores";

const categoryColor: Record<NotificationCategory, string> = {
  CRITICAL: "bg-red-100 text-red-700",
  WARNING: "bg-amber-100 text-amber-700",
  INFO: "bg-blue-100 text-blue-700",
  MARKETING: "bg-purple-100 text-purple-700",
};

const CATEGORIES: NotificationCategory[] = [
  "CRITICAL",
  "WARNING",
  "INFO",
  "MARKETING",
];

const CHANNELS: NotificationChannel[] = [
  "IN_APP",
  "EMAIL",
  "TELEGRAM",
  "PUSH",
  "SMS",
];

/** Extract ``{{variable}}`` placeholders from a template string. */
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
  category: NotificationCategory;
  channel: NotificationChannel;
  subject: string;
  bodyTemplate: string;
}

const EMPTY_DRAFT: DraftTemplate = {
  code: "",
  category: "INFO",
  channel: "EMAIL",
  subject: "",
  bodyTemplate: "",
};

export default function NotificationTemplatesPage() {
  const { templates, add, update, remove, reset } =
    useNotificationTemplatesStore();

  const [hydrated, setHydrated] = useState(false);
  useEffect(() => setHydrated(true), []);
  const list: NotificationTemplate[] = hydrated ? templates : [];

  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draft, setDraft] = useState<DraftTemplate>(EMPTY_DRAFT);

  function startCreate() {
    setEditingId(null);
    setDraft(EMPTY_DRAFT);
    setOpen(true);
  }

  function startEdit(t: NotificationTemplate) {
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
    const variables = Array.from(new Set([...subjectVars, ...bodyVars]));
    if (editingId) {
      update(editingId, { ...draft, variables });
    } else {
      add({ ...draft, variables });
    }
    setOpen(false);
  }

  function sendTest(t: NotificationTemplate) {
    // Hook: POST to notification-service when admin-gateway is built.
    // For now we just open the user's mail client with a prefilled body.
    if (typeof window === "undefined") return;
    const vars = Object.fromEntries(
      t.variables.map((v) => [v, `[${v}]`]),
    );
    let body = t.bodyTemplate;
    for (const [k, v] of Object.entries(vars)) {
      body = body.replaceAll(`{{${k}}}`, v);
    }
    // In a real implementation, this would POST to
    // /api/v1/admin/notifications/test-send with the template id.
    // eslint-disable-next-line no-console
    console.info("[templates] test-send", { code: t.code, body });
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bell className="w-6 h-6" />
            Notification Templates
          </h1>
          <p className="text-gray-500">
            Manage templates for system notifications and broadcasts. State
            persists in localStorage until the admin-gateway backend is
            wired up (docs/15-roadmap §2 Phase 2).
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={reset} title="Reset to defaults">
            <RotateCcw className="w-4 h-4" />
          </Button>
          <Button onClick={startCreate}>
            <Plus className="w-4 h-4 mr-2" />
            New Template
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {list.map((t) => (
          <Card key={t.id} className="hover:border-primary transition-colors">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <CardTitle className="text-base">{t.code}</CardTitle>
                    <Badge className={categoryColor[t.category]}>
                      {t.category}
                    </Badge>
                    <Badge variant="outline">{t.channel}</Badge>
                  </div>
                  <CardDescription className="mt-1">
                    Subject: {t.subject}
                  </CardDescription>
                </div>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => startEdit(t)}
                    aria-label="Edit"
                  >
                    <Edit2 className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => sendTest(t)}
                    aria-label="Send test"
                  >
                    <Send className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => remove(t.id)}
                    aria-label="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground bg-muted/30 p-3 rounded-md font-mono whitespace-pre-wrap">
                {t.bodyTemplate}
              </div>
              <div className="mt-3 flex flex-wrap gap-1">
                <span className="text-xs text-muted-foreground">
                  Variables:
                </span>
                {t.variables.map((v, idx) => (
                  <span
                    key={idx}
                    className="text-xs font-mono bg-blue-50 text-blue-700 px-2 py-0.5 rounded"
                  >
                    {`{{${v}}}`}
                  </span>
                ))}
                {t.variables.length === 0 && (
                  <span className="text-xs text-muted-foreground italic">
                    (none)
                  </span>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
        {list.length === 0 && (
          <div className="col-span-2 text-center py-12 text-muted-foreground">
            No notification templates. Create one to get started.
          </div>
        )}
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingId ? "Edit Template" : "New Notification Template"}
            </DialogTitle>
            <DialogDescription>
              Use <code className="font-mono">{`{{variable_name}}`}</code>{" "}
              placeholders in the subject and body. Variables are
              auto-extracted on save.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="tpl-code">Code</Label>
              <Input
                id="tpl-code"
                value={draft.code}
                onChange={(e) =>
                  setDraft({ ...draft, code: e.target.value })
                }
                placeholder="e.g. billing.invoice_failed"
                disabled={editingId !== null}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="tpl-cat">Category</Label>
                <select
                  id="tpl-cat"
                  className="w-full border rounded-md px-3 py-2 mt-1"
                  value={draft.category}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      category: e.target.value as NotificationCategory,
                    })
                  }
                >
                  {CATEGORIES.map((c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <Label htmlFor="tpl-channel">Channel</Label>
                <select
                  id="tpl-channel"
                  className="w-full border rounded-md px-3 py-2 mt-1"
                  value={draft.channel}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      channel: e.target.value as NotificationChannel,
                    })
                  }
                >
                  {CHANNELS.map((c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div>
              <Label htmlFor="tpl-subject">Subject</Label>
              <Input
                id="tpl-subject"
                value={draft.subject}
                onChange={(e) =>
                  setDraft({ ...draft, subject: e.target.value })
                }
                placeholder="Email subject line"
              />
            </div>
            <div>
              <Label htmlFor="tpl-body">Body</Label>
              <Textarea
                id="tpl-body"
                value={draft.bodyTemplate}
                onChange={(e) =>
                  setDraft({ ...draft, bodyTemplate: e.target.value })
                }
                placeholder="Template body — supports {{variables}}"
                rows={6}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={submit}
              disabled={!draft.code.trim() || !draft.subject.trim()}
            >
              {editingId ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
