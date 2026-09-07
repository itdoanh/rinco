// TypeScript Connect-RPC client for the auth-service (AuthService).
//
//   import { createPromiseClient } from "@bufbuild/connect";
//   import { createConnectTransport } from "@bufbuild/connect-web";
//   import { AuthService } from "../gen/auth/v1/auth_connect";
//
//   const client = createPromiseClient(AuthService, createConnectTransport({
//     baseUrl: "http://localhost:8081",
//   }));
//   const res = await client.validateToken({ accessToken });
//
// The shape below mirrors the Connect-RPC schema generated from
// services/auth-service/proto/auth/v1/auth.proto (manually maintained
// until buf is wired in CI).
export interface ValidateTokenRequest {
  accessToken: string;
}
export interface ValidateTokenResponse {
  valid: boolean;
  userId?: string;
  tenantId?: string;
  roles?: string[];
  scope?: string;
  expiresAt?: string;
}
export interface LoginRequest {
  email: string;
  password: string;
  tenantSlug?: string;
}
export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
  userId?: string;
  tenantId?: string;
}
export interface GetUserRequest { userId: string; }
export interface GetUserResponse {
  id: string;
  tenantId: string;
  email: string;
  fullName?: string;
  status: string;
  isSuperAdmin?: boolean;
}

// Plain fetch wrappers — no codegen required.  Use them from any TS / JS
// runtime (Node 18+, browser, Deno, Bun).
const DEFAULT_BASE = process.env.AUTH_BASE_URL ?? "http://localhost:8081";

export class AuthClient {
  constructor(private readonly base: string = DEFAULT_BASE) {}

  async login(req: LoginRequest): Promise<LoginResponse> {
    return this.rpc<LoginResponse>("auth.v1.AuthService", "Login", req);
  }
  async validateToken(req: ValidateTokenRequest): Promise<ValidateTokenResponse> {
    return this.rpc<ValidateTokenResponse>("auth.v1.AuthService", "ValidateToken", req);
  }
  async getUser(req: GetUserRequest): Promise<GetUserResponse> {
    return this.rpc<GetUserResponse>("auth.v1.AuthService", "GetUser", req);
  }

  private async rpc<T>(svc: string, method: string, body: unknown): Promise<T> {
    const r = await fetch(`${this.base}/internal/${svc}/${method}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!r.ok) throw new Error(`rpc ${svc}/${method} ${r.status}: ${await r.text()}`);
    return r.json() as Promise<T>;
  }
}

// Smoke test when invoked directly:
//   npx tsx examples/grpc-client.ts
if (typeof require !== "undefined" && require.main === module) {
  (async () => {
    const c = new AuthClient();
    const tok = await c.login({ email: "alice@example.com", password: "correct horse battery staple" });
    console.log("access:", tok.accessToken.slice(0, 24) + "…");
    const v = await c.validateToken({ accessToken: tok.accessToken });
    console.log("valid:", v.valid, "user:", v.userId, "tenant:", v.tenantId);
  })();
}
