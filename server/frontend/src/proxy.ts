import { withAuth } from "next-auth/middleware";
import { NextResponse } from "next/server";

export default withAuth(
  function proxy(req) {
    const token = req.nextauth.token;
    const isAdminPath = req.nextUrl.pathname.startsWith("/admin");

    // If it's an admin path and user doesn't have admin role, redirect to dashboard
    if (isAdminPath && !token?.isAdmin) {
      return NextResponse.redirect(new URL("/dashboard", req.url));
    }

    return NextResponse.next();
  },
  {
    callbacks: {
      // The middleware only runs if authorized returns true
      authorized: ({ token }) => !!token,
    },
    pages: {
      signIn: "/login",
    },
  }
);

// Only apply middleware to protected routes
export const config = {
  matcher: ["/dashboard/:path*", "/admin/:path*"],
};
