import type { NextAuthOptions, Session } from "next-auth";
import type { JWT } from "next-auth/jwt";
import CredentialsProvider from "next-auth/providers/credentials";

type KeycloakAccessTokenClaims = {
  realm_access?: {
    roles?: string[];
  };
  resource_access?: Record<
    string,
    {
      roles?: string[];
    }
  >;
  sub?: string;
  preferred_username?: string;
  email?: string;
  name?: string;
};

type KeycloakTokenResponse = {
  access_token: string;
  expires_in: number;
  refresh_expires_in: number;
  refresh_token: string;
  token_type: string;
  id_token?: string;
  "not-before-policy": number;
  session_state: string;
  scope: string;
};

interface KeycloakUser {
  id: string;
  name: string;
  email: string;
  accessToken: string;
  refreshToken: string;
  idToken?: string;
  expiresIn: number;
}

const keycloakClientId = process.env.KEYCLOAK_CLIENT_ID || "termviewer-app";
const keycloakClientSecret = process.env.KEYCLOAK_CLIENT_SECRET || "";
export const keycloakAdminRole = process.env.KEYCLOAK_ADMIN_ROLE || "termviewer-admin";
const frontendBaseUrl = (process.env.NEXTAUTH_URL || "https://termviewer.local").replace(/\/$/, "");
const keycloakIssuer = process.env.KEYCLOAK_ISSUER?.replace(/\/$/, "");

function decodeAccessTokenClaims(accessToken?: string | null): KeycloakAccessTokenClaims | null {
  if (!accessToken) return null;
  const parts = accessToken.split(".");
  if (parts.length < 2) return null;
  try {
    const payload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const json = Buffer.from(payload, "base64").toString("utf-8");
    return JSON.parse(json) as KeycloakAccessTokenClaims;
  } catch {
    return null;
  }
}

function extractTokenRoles(accessToken?: string | null): string[] {
  const claims = decodeAccessTokenClaims(accessToken);
  if (!claims) return [];
  const roles = new Set<string>();
  (claims.realm_access?.roles ?? []).forEach(r => roles.add(r));
  (claims.resource_access?.[keycloakClientId]?.roles ?? []).forEach(r => roles.add(r));
  return Array.from(roles);
}

export function sessionHasAdminRole(session: Session | null | undefined): boolean {
  return Boolean(session?.isAdmin);
}

function buildLogoutUrl(idToken?: string | null): string | undefined {
  if (!keycloakIssuer) return undefined;
  const url = new URL(`${keycloakIssuer}/protocol/openid-connect/logout`);
  url.searchParams.set("post_logout_redirect_uri", `${frontendBaseUrl}/`);
  url.searchParams.set("client_id", keycloakClientId);
  if (idToken) url.searchParams.set("id_token_hint", idToken);
  return url.toString();
}

async function refreshAccessToken(token: JWT): Promise<JWT> {
  if (!keycloakIssuer || !token.refreshToken) {
    return { ...token, error: "RefreshAccessTokenError" };
  }

  try {
    const params = new URLSearchParams({
      client_id: keycloakClientId,
      grant_type: "refresh_token",
      refresh_token: token.refreshToken as string,
      scope: "openid profile email",
    });
    if (keycloakClientSecret) params.set("client_secret", keycloakClientSecret);

    const response = await fetch(`${keycloakIssuer}/protocol/openid-connect/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: params.toString(),
    });

    if (!response.ok) throw new Error("Failed to refresh token");

    const refreshed = (await response.json()) as KeycloakTokenResponse;
    const nextRoles = extractTokenRoles(refreshed.access_token);

    return {
      ...token,
      accessToken: refreshed.access_token,
      accessTokenExpiresAt: Date.now() + refreshed.expires_in * 1000,
      refreshToken: refreshed.refresh_token || token.refreshToken,
      idToken: refreshed.id_token || token.idToken,
      roles: nextRoles,
      isAdmin: nextRoles.includes(keycloakAdminRole),
      logoutUrl: buildLogoutUrl(refreshed.id_token || (token.idToken as string)),
    };
  } catch {
    return { ...token, error: "RefreshAccessTokenError" };
  }
}

export const authOptions: NextAuthOptions = {
  pages: { signIn: "/login" },
  session: { strategy: "jwt" },
  // Harden cookies for Production
  useSecureCookies: process.env.NODE_ENV === "production",
  cookies: {
    sessionToken: {
      name: process.env.NODE_ENV === "production" ? `__Secure-next-auth.session-token` : `next-auth.session-token`,
      options: {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: process.env.NODE_ENV === "production",
      },
    },
  },
  providers: [
    CredentialsProvider({
      name: "Keycloak",
      credentials: {
        username: { label: "Username", type: "text" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        if (!credentials?.username || !credentials?.password || !keycloakIssuer) return null;

        const params = new URLSearchParams({
          client_id: keycloakClientId,
          grant_type: "password",
          username: credentials.username,
          password: credentials.password,
          scope: "openid profile email",
        });
        if (keycloakClientSecret) params.set("client_secret", keycloakClientSecret);

        try {
          const response = await fetch(`${keycloakIssuer}/protocol/openid-connect/token`, {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: params.toString(),
          });

          if (!response.ok) {
            return null;
          }

          const tokens = (await response.json()) as KeycloakTokenResponse;
          const claims = decodeAccessTokenClaims(tokens.access_token);

          // Return an object that includes the tokens
          return {
            id: claims?.sub || credentials.username,
            name: claims?.name || claims?.preferred_username || credentials.username,
            email: claims?.email || "",
            accessToken: tokens.access_token,
            refreshToken: tokens.refresh_token,
            idToken: tokens.id_token,
            expiresIn: tokens.expires_in,
          } as KeycloakUser;
        } catch {
          return null;
        }
      },
    }),
  ],
  callbacks: {
    async jwt({ token, user }) {
      // Initial sign in
      if (user) {
        const u = user as KeycloakUser;
        const nextRoles = extractTokenRoles(u.accessToken);

        token.accessToken = u.accessToken;
        token.refreshToken = u.refreshToken;
        token.idToken = u.idToken;
        token.accessTokenExpiresAt = Date.now() + (u.expiresIn || 300) * 1000;
        token.roles = nextRoles;
        token.isAdmin = nextRoles.includes(keycloakAdminRole);
        token.logoutUrl = buildLogoutUrl(u.idToken);
        token.sub = u.id;
        token.name = u.name;
        token.email = u.email;
        return token;
      }

      // Check if token expired
      if (typeof token.accessTokenExpiresAt === "number" && Date.now() < token.accessTokenExpiresAt - 30000) {
        return token;
      }

      return refreshAccessToken(token);
    },
    async session({ session, token }) {
      session.accessToken = token.accessToken as string;
      session.idToken = token.idToken as string;
      session.roles = (token.roles as string[]) || [];
      session.isAdmin = !!token.isAdmin;
      session.logoutUrl = token.logoutUrl as string;
      session.error = token.error as "RefreshAccessTokenError" | undefined;

      if (session.user) {
        session.user.id = token.sub as string;
        session.user.name = token.name as string;
        session.user.email = token.email as string;
      }

      return session;
    },
  },
};
