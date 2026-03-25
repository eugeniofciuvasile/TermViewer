import type { DefaultSession } from "next-auth"

declare module "next-auth" {
  interface Session {
    accessToken?: string
    error?: "RefreshAccessTokenError"
    idToken?: string
    logoutUrl?: string
    roles: string[]
    isAdmin: boolean
    user: {
      id?: string
    } & DefaultSession["user"]
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string
    accessTokenExpiresAt?: number
    error?: "RefreshAccessTokenError"
    idToken?: string
    logoutUrl?: string
    refreshToken?: string
    roles: string[]
    isAdmin: boolean
  }
}
