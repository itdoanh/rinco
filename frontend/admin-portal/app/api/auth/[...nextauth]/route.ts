import NextAuth from "next-auth";
import { authOptions } from "@/lib/auth";

// NextAuth v5 returns { handlers: { GET, POST }, auth, signIn, signOut }.
// Re-export handlers directly so Next.js can route GET/POST requests.
const { handlers } = NextAuth(authOptions);
export const { GET, POST } = handlers;
