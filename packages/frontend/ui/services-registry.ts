/**
 * RINCO service registry — single source of truth cho port mapping
 * của mọi backend service. Mỗi frontend app đọc từ file này để biết
 * URL/WS endpoint của từng service.
 *
 * Canonical port mapping (sau WS-A):
 *   auth            8081
 *   tenant          8082
 *   crm             8083
 *   dynamic-model   8084
 *   lead            8085
 *   landing         8086
 *   email           8087
 *   notification    8088
 *   ai-sre          8090
 *   lead-scoring    8091
 *   rag-chatbot     8092
 *   recording       8093
 *   stt             8094
 *   billing         8095
 *   observability   8096
 *   search          8097
 *   meta-capi       8098
 *   chat-engine     8101 (WS)
 *   webrtc-sfu      8102 (WS)
 *
 * Override trong dev bằng biến môi trường NEXT_PUBLIC_SVC_<name>.
 */

export type ServiceName =
  | "auth"
  | "tenant"
  | "crm"
  | "dynamic-model"
  | "lead"
  | "landing"
  | "email"
  | "notification"
  | "ai-sre"
  | "lead-scoring"
  | "rag-chatbot"
  | "recording"
  | "stt"
  | "billing"
  | "observability"
  | "search"
  | "meta-capi";

export type WebSocketService = "chat-engine" | "webrtc-sfu";

interface ServiceEntry {
  /** Tên service (snake-case cho env override) */
  name: ServiceName | WebSocketService;
  /** HTTP port (cho REST) */
  port: number;
  /** WebSocket port (cho realtime services; nếu khác HTTP) */
  wsPort?: number;
  /** Mô tả ngắn */
  description: string;
}

export const SERVICES: ReadonlyArray<ServiceEntry> = [
  { name: "auth", port: 8081, description: "Authentication / session / users" },
  { name: "tenant", port: 8082, description: "Tenant management / branding / domains" },
  { name: "crm", port: 8083, description: "Contacts / deals / activities / RBAC" },
  { name: "dynamic-model", port: 8084, description: "Feature flags / dynamic schemas" },
  { name: "lead", port: 8085, description: "Lead capture / scoring / CAPI events" },
  { name: "landing", port: 8086, description: "Landing pages / CMS / forms" },
  { name: "email", port: 8087, description: "Transactional email / templates" },
  { name: "notification", port: 8088, description: "In-app / push / Telegram notifications" },
  { name: "ai-sre", port: 8090, description: "AI-driven SRE assistant" },
  { name: "lead-scoring", port: 8091, description: "ML lead scoring service" },
  { name: "rag-chatbot", port: 8092, description: "RAG chatbot / vector search" },
  { name: "recording", port: 8093, description: "Meeting recording storage" },
  { name: "stt", port: 8094, description: "Speech-to-text transcription" },
  { name: "billing", port: 8095, description: "Subscriptions / invoices / billing" },
  { name: "observability", port: 8096, description: "Metrics / logs / traces" },
  { name: "search", port: 8097, description: "Full-text + vector search" },
  { name: "meta-capi", port: 8098, description: "Meta Conversions API gateway" },
  { name: "chat-engine", port: 8101, wsPort: 8101, description: "WebSocket chat / meeting signaling" },
  { name: "webrtc-sfu", port: 8102, wsPort: 8102, description: "WebRTC SFU media server" },
] as const;

/** Map service name → ServiceEntry. */
const SERVICE_MAP: Record<string, ServiceEntry> = SERVICES.reduce(
  (acc, s) => {
    acc[s.name] = s;
    return acc;
  },
  {} as Record<string, ServiceEntry>,
);

/** Lấy port override từ env (NEXT_PUBLIC_SVC_<NAME>). */
function readPortOverride(name: string, fallback: number): number {
  if (typeof process === "undefined") return fallback;
  const v = process.env[`NEXT_PUBLIC_SVC_${name.toUpperCase().replace(/-/g, "_")}`];
  if (!v) return fallback;
  const n = Number(v);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

/** Trả về HTTP base URL cho service. */
export function httpUrl(service: ServiceName): string {
  const entry = SERVICE_MAP[service];
  if (!entry) throw new Error(`Unknown RINCO service: ${service}`);
  const port = readPortOverride(entry.name, entry.port);
  return `http://localhost:${port}`;
}

/** Trả về WebSocket URL cho service. */
export function wsUrl(service: WebSocketService): string {
  const entry = SERVICE_MAP[service];
  if (!entry) throw new Error(`Unknown RINCO WS service: ${service}`);
  const port = readPortOverride(entry.name, entry.wsPort ?? entry.port);
  return `ws://localhost:${port}`;
}

/**
 * Build a path URL cho service: prepend base URL + endpoint.
 * Endpoint nên bắt đầu bằng "/" (e.g. "/v1/leads").
 */
export function serviceUrl(service: ServiceName, endpoint: string): string {
  const base = httpUrl(service).replace(/\/$/, "");
  const path = endpoint.startsWith("/") ? endpoint : `/${endpoint}`;
  return `${base}${path}`;
}

/** Trả về thông tin service (cho debug / health page). */
export function getServiceInfo(name: string): ServiceEntry | undefined {
  return SERVICE_MAP[name];
}

/** List toàn bộ service (cho health endpoint page). */
export function listServices(): ReadonlyArray<ServiceEntry> {
  return SERVICES;
}
