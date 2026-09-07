import NextAuth, { NextAuthOptions, User } from 'next-auth';
import CredentialsProvider from 'next-auth/providers/credentials';

// Extended user type
interface ExtendedUser extends User {
  role?: string;
  tenant_id?: string;
}

// NextAuth configuration options
export const authOptions: NextAuthOptions = {
  providers: [
    CredentialsProvider({
      name: 'Credentials',
      credentials: {
        email: { label: 'Email', type: 'email' },
        password: { label: 'Password', type: 'password' },
      },
      async authorize(credentials): Promise<ExtendedUser | null> {
        if (!credentials?.email || !credentials?.password) {
          return null;
        }

        try {
          // Call the auth service API
          const response = await fetch(process.env.NEXT_PUBLIC_API_URL + '/api/auth/login', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              email: credentials.email,
              password: credentials.password,
            }),
          });

          if (!response.ok) {
            return null;
          }

          const data = await response.json();

          // Return user object
          return {
            id: data.user.id,
            email: data.user.email,
            name: data.user.name,
            role: data.user.role,
            tenant_id: data.user.tenant_id,
          };
        } catch (error) {
          console.error('Auth error:', error);
          return null;
        }
      },
    }),
  ],

  callbacks: {
    async jwt({ token, user }) {
      if (user) {
        token.id = user.id;
        token.role = (user as ExtendedUser).role;
        token.tenant_id = (user as ExtendedUser).tenant_id;
      }
      return token;
    },

    async session({ session, token }) {
      if (session.user) {
        (session.user as ExtendedUser).id = token.id as string;
        (session.user as ExtendedUser).role = token.role as string;
        (session.user as ExtendedUser).tenant_id = token.tenant_id as string;
      }
      return session;
    },
  },

  pages: {
    signIn: '/(auth)/login',
    error: '/(auth)/login',
  },

  session: {
    strategy: 'jwt',
    maxAge: 24 * 60 * 60, // 24 hours
  },

  secret: process.env.NEXTAUTH_SECRET || 'development-secret-key-change-in-production',
};

// Helper to check if user has required role
export function hasRole(userRole: string | undefined, requiredRoles: string[]): boolean {
  if (!userRole) return false;
  return requiredRoles.includes(userRole);
}

// Helper to check if user has admin access
export function isAdmin(role: string | undefined): boolean {
  return role === 'admin' || role === 'super_admin';
}

// Helper to check if user has access to tenant
export function hasTenantAccess(userTenantId: string | undefined, targetTenantId: string, role: string | undefined): boolean {
  if (isAdmin(role)) return true;
  return userTenantId === targetTenantId;
}

// Role hierarchy for permission checks
export const roleHierarchy: Record<string, number> = {
  viewer: 1,
  user: 2,
  manager: 3,
  admin: 4,
  super_admin: 5,
};

export function hasMinimumRole(userRole: string | undefined, minimumRole: string): boolean {
  if (!userRole) return false;
  const userLevel = roleHierarchy[userRole] || 0;
  const requiredLevel = roleHierarchy[minimumRole] || 0;
  return userLevel >= requiredLevel;
}
