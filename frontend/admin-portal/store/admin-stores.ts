/**
 * Admin store slice for feature flags, notification templates, and quorum
 * requests.  These three subsystems are managed by the Super Admin
 * Portal and do not yet have a dedicated ``admin-gateway`` backend
 * service (deferred per docs/15-roadmap §2 Phase 2).
 *
 * Until the gateway is built we persist the state in ``localStorage`` so
 * that admins can edit it and have the changes survive a reload.  Once
 * the gateway is available, replace ``load``/``save`` with calls to the
 * new HTTP endpoints and the rest of the application will continue to
 * work unchanged.
 */
import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

// =============================================================================
// Feature Flags
// =============================================================================

export type FeatureFlagCategory =
  | "core"
  | "experimental"
  | "beta"
  | "deprecated";

export interface FeatureFlag {
  key: string;
  description: string;
  enabled: boolean;
  rolloutPercentage: number;
  category: FeatureFlagCategory;
  lastModified: string;
}

const SEED_FLAGS: FeatureFlag[] = [
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
];

interface FeatureFlagsState {
  flags: FeatureFlag[];
  toggle: (key: string) => void;
  updateRollout: (key: string, percentage: number) => void;
  add: (flag: Omit<FeatureFlag, "lastModified">) => void;
  remove: (key: string) => void;
  reset: () => void;
  isEnabled: (key: string) => boolean;
}

export const useFeatureFlagsStore = create<FeatureFlagsState>()(
  persist(
    (set, get) => ({
      flags: SEED_FLAGS,
      toggle: (key) =>
        set((state) => ({
          flags: state.flags.map((f) =>
            f.key === key
              ? {
                  ...f,
                  enabled: !f.enabled,
                  rolloutPercentage: !f.enabled ? Math.max(f.rolloutPercentage, 1) : 0,
                  lastModified: new Date().toISOString(),
                }
              : f,
          ),
        })),
      updateRollout: (key, percentage) =>
        set((state) => ({
          flags: state.flags.map((f) =>
            f.key === key
              ? {
                  ...f,
                  rolloutPercentage: Math.max(0, Math.min(100, percentage)),
                  lastModified: new Date().toISOString(),
                }
              : f,
          ),
        })),
      add: (flag) =>
        set((state) => ({
          flags: [
            ...state.flags,
            { ...flag, lastModified: new Date().toISOString() },
          ],
        })),
      remove: (key) =>
        set((state) => ({ flags: state.flags.filter((f) => f.key !== key) })),
      reset: () => set({ flags: SEED_FLAGS }),
      isEnabled: (key) => {
        const f = get().flags.find((x) => x.key === key);
        return f?.enabled ?? false;
      },
    }),
    {
      name: "rinco-admin-feature-flags",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);

// =============================================================================
// Notification Templates
// =============================================================================

export type NotificationCategory =
  | "CRITICAL"
  | "WARNING"
  | "INFO"
  | "MARKETING";

export type NotificationChannel =
  | "IN_APP"
  | "EMAIL"
  | "TELEGRAM"
  | "PUSH"
  | "SMS";

export interface NotificationTemplate {
  id: string;
  code: string;
  category: NotificationCategory;
  channel: NotificationChannel;
  subject: string;
  bodyTemplate: string;
  variables: string[];
}

const SEED_TEMPLATES: NotificationTemplate[] = [
  {
    id: "1",
    code: "maintenance.scheduled",
    category: "WARNING",
    channel: "EMAIL",
    subject: "Scheduled Maintenance on {{date}}",
    bodyTemplate:
      "Hi {{tenant_name}}, we have scheduled maintenance on {{date}} at {{time}}. Estimated downtime: {{duration}}.",
    variables: ["tenant_name", "date", "time", "duration"],
  },
  {
    id: "2",
    code: "billing.invoice_failed",
    category: "CRITICAL",
    channel: "EMAIL",
    subject: "Payment Failed for Invoice {{invoice_number}}",
    bodyTemplate:
      "Your payment for {{plan}} plan failed. Please update payment method within 7 days.",
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
];

interface NotificationTemplatesState {
  templates: NotificationTemplate[];
  add: (template: Omit<NotificationTemplate, "id">) => void;
  update: (id: string, updates: Partial<NotificationTemplate>) => void;
  remove: (id: string) => void;
  reset: () => void;
  findByCode: (code: string) => NotificationTemplate | undefined;
}

export const useNotificationTemplatesStore = create<NotificationTemplatesState>()(
  persist(
    (set, get) => ({
      templates: SEED_TEMPLATES,
      add: (template) =>
        set((state) => ({
          templates: [
            ...state.templates,
            { ...template, id: `tpl-${Date.now()}` },
          ],
        })),
      update: (id, updates) =>
        set((state) => ({
          templates: state.templates.map((t) =>
            t.id === id ? { ...t, ...updates } : t,
          ),
        })),
      remove: (id) =>
        set((state) => ({
          templates: state.templates.filter((t) => t.id !== id),
        })),
      reset: () => set({ templates: SEED_TEMPLATES }),
      findByCode: (code) => get().templates.find((t) => t.code === code),
    }),
    {
      name: "rinco-admin-notification-templates",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);

// =============================================================================
// Quorum (Multi-Party Authorization) requests
// =============================================================================

export type QuorumStatus = "PENDING" | "APPROVED" | "REJECTED" | "EXPIRED";

export interface QuorumRequest {
  id: string;
  action: string;
  reason: string;
  initiator: string;
  requiredSigs: number;
  collectedSigs: string[];
  status: QuorumStatus;
  expiresAt: string;
  createdAt: string;
}

const now = Date.now();

const SEED_QUORUMS: QuorumRequest[] = [
  {
    id: "qr-1",
    action: "tenant.delete",
    reason: "GDPR Right to be Forgotten request from customer (verified)",
    initiator: "security@rinco.app",
    requiredSigs: 2,
    collectedSigs: ["security@rinco.app", "owner@rinco.app"],
    status: "APPROVED",
    expiresAt: new Date(now - 3600_000).toISOString(),
    createdAt: new Date(now - 7200_000).toISOString(),
  },
  {
    id: "qr-2",
    action: "gateway.global_config",
    reason: "Emergency rate limit increase for upcoming marketing campaign",
    initiator: "ops@rinco.app",
    requiredSigs: 2,
    collectedSigs: ["ops@rinco.app"],
    status: "PENDING",
    expiresAt: new Date(now + 1800_000).toISOString(),
    createdAt: new Date(now - 600_000).toISOString(),
  },
  {
    id: "qr-3",
    action: "tenant.lock",
    reason: "Suspicious billing activity detected, locking for investigation",
    initiator: "finance@rinco.app",
    requiredSigs: 2,
    collectedSigs: [],
    status: "PENDING",
    expiresAt: new Date(now + 3000_000).toISOString(),
    createdAt: new Date(now - 120_000).toISOString(),
  },
];

interface QuorumState {
  requests: QuorumRequest[];
  create: (
    req: Omit<QuorumRequest, "id" | "createdAt" | "status" | "collectedSigs"> & {
      collectedSigs?: string[];
    },
  ) => QuorumRequest;
  sign: (id: string, signer: string) => void;
  reject: (id: string, signer: string) => void;
  /** Mark any expired PENDING requests as EXPIRED.  Called lazily on
   *  store reads via :func:`useQuorumStore.tickExpiry`. */
  tickExpiry: () => void;
  reset: () => void;
}

export const useQuorumStore = create<QuorumState>()(
  persist(
    (set, get) => ({
      requests: SEED_QUORUMS,
      create: (req) => {
        const created: QuorumRequest = {
          ...req,
          collectedSigs: req.collectedSigs ?? [req.initiator],
          id: `qr-${Date.now()}`,
          createdAt: new Date().toISOString(),
          status: "PENDING",
        };
        set((state) => ({ requests: [created, ...state.requests] }));
        return created;
      },
      sign: (id, signer) =>
        set((state) => ({
          requests: state.requests.map((q) => {
            if (q.id !== id || q.status !== "PENDING") return q;
            if (q.collectedSigs.includes(signer)) return q;
            if (new Date(q.expiresAt).getTime() < Date.now()) {
              return { ...q, status: "EXPIRED" };
            }
            const collected = [...q.collectedSigs, signer];
            return {
              ...q,
              collectedSigs: collected,
              status:
                collected.length >= q.requiredSigs ? "APPROVED" : q.status,
            };
          }),
        })),
      reject: (id, _signer) =>
        set((state) => ({
          requests: state.requests.map((q) =>
            q.id === id && q.status === "PENDING"
              ? { ...q, status: "REJECTED" }
              : q,
          ),
        })),
      tickExpiry: () =>
        set((state) => ({
          requests: state.requests.map((q) => {
            if (
              q.status === "PENDING" &&
              new Date(q.expiresAt).getTime() < Date.now()
            ) {
              return { ...q, status: "EXPIRED" };
            }
            return q;
          }),
        })),
      reset: () => set({ requests: SEED_QUORUMS }),
    }),
    {
      name: "rinco-admin-quorum",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);
