import axios from "axios";
import { getSession, signOut } from "next-auth/react";

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_BACKEND_URL || process.env.NEXT_PUBLIC_API_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// Request Interceptor: Automatically add Bearer token
api.interceptors.request.use(async (config) => {
  const session = await getSession();
  if (session?.accessToken) {
    config.headers.Authorization = `Bearer ${session.accessToken}`;
  }
  return config;
});

// Response Interceptor: Handle 401s and sanitize errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // If we get a 401, the token is likely expired or invalid
    if (error.response?.status === 401) {
      console.warn("Unauthorized access detected...");
      if (typeof window !== "undefined") {
        signOut({ callbackUrl: "/login?error=SessionExpired" });
      }
    }

    // Sanitize error before returning to component to prevent leakage
    const message = error.response?.data?.error || "An unexpected error occurred";
    return Promise.reject(new Error(message));
  }
);

export default api;
