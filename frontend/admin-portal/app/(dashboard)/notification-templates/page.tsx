"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Bell, Plus, Edit2, Trash2, Send } from "lucide-react";

interface Template {
  id: string
  code: string
  category: "CRITICAL" | "WARNING" | "INFO" | "MARKETING"
  channel: "IN_APP" | "EMAIL" | "TELEGRAM" | "PUSH" | "SMS"
  subject: string
  bodyTemplate: string
  variables: string[]
}

const sampleTemplates: Template[] = [
  {
    id: "1",
    code: "maintenance.scheduled",
    category: "WARNING",
    channel: "EMAIL",
    subject: "Scheduled Maintenance on {{date}}",
    bodyTemplate: "Hi {{tenant_name}}, we have scheduled maintenance on {{date}} at {{time}}. Estimated downtime: {{duration}}.",
    variables: ["tenant_name", "date", "time", "duration"],
  },
  {
    id: "2",
    code: "billing.invoice_failed",
    category: "CRITICAL",
    channel: "EMAIL",
    subject: "Payment Failed for Invoice {{invoice_number}}",
    bodyTemplate: "Your payment for {{plan}} plan failed. Please update payment method within 7 days.",
    variables: ["invoice_number", "plan"],
  },
  {
    id: "3",
    code: "welcome.new_tenant",
    category: "INFO",
    channel: "IN_APP",
    subject: "Welcome to RINCO, {{tenant_name}}!",
    bodyTemplate: "Get started by configuring your team structure.",
    variables: ["tenant_name"],
  },
  {
    id: "4",
    code: "feature.new_release",
    category: "MARKETING",
    channel: "EMAIL",
    subject: "New: {{feature_name}}",
    bodyTemplate: "We've just released {{feature_name}}. {{description}}",
    variables: ["feature_name", "description"],
  },
]

const categoryColor = {
  CRITICAL: "bg-red-100 text-red-700",
  WARNING: "bg-amber-100 text-amber-700",
  INFO: "bg-blue-100 text-blue-700",
  MARKETING: "bg-purple-100 text-purple-700",
}

export default function NotificationTemplatesPage() {
  const [templates, setTemplates] = useState(sampleTemplates)

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Bell className="w-6 h-6" />
            Notification Templates
          </h1>
          <p className="text-gray-500">
            Manage templates for system notifications and broadcasts.
          </p>
        </div>
        <Button>
          <Plus className="w-4 h-4 mr-2" />
          New Template
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {templates.map(t => (
          <Card key={t.id} className="hover:border-primary transition-colors">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <CardTitle className="text-base">{t.code}</CardTitle>
                    <Badge className={categoryColor[t.category]}>{t.category}</Badge>
                    <Badge variant="outline">{t.channel}</Badge>
                  </div>
                  <CardDescription className="mt-1">
                    Subject: {t.subject}
                  </CardDescription>
                </div>
                <div className="flex gap-1">
                  <Button variant="ghost" size="sm">
                    <Edit2 className="w-4 h-4" />
                  </Button>
                  <Button variant="ghost" size="sm">
                    <Send className="w-4 h-4" />
                  </Button>
                  <Button variant="ghost" size="sm">
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground bg-muted/30 p-3 rounded-md font-mono">
                {t.bodyTemplate}
              </div>
              <div className="mt-3 flex flex-wrap gap-1">
                <span className="text-xs text-muted-foreground">Variables:</span>
                {t.variables.map((v, idx) => (
                  <span key={idx} className="text-xs font-mono bg-blue-50 text-blue-700 px-2 py-0.5 rounded">
                    {`{{${v}}}`}
                  </span>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
