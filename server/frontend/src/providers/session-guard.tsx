"use client";

import { useEffect } from "react";
import { useSession, signOut } from "next-auth/react";

export default function SessionGuard({ children }: { children: React.ReactNode }) {
  const { data: session } = useSession();

  useEffect(() => {
    // If the session returns a RefreshAccessTokenError, it means the 
    // Keycloak refresh token has expired and we MUST sign out.
    if (session?.error === "RefreshAccessTokenError") {
      console.warn("Session expired (Refresh token invalid). Signing out...");
      signOut({ callbackUrl: "/login?error=SessionExpired" });
    }
  }, [session]);

  return <>{children}</>;
}
