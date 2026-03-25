"use client";

import { signIn } from "next-auth/react";
import { useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";
import { ArrowRight, LoaderCircle, Shield, Terminal } from "lucide-react";
import Link from "next/link";

function LoginForm() {
  const searchParams = useSearchParams();
  const callbackUrl = searchParams.get("callbackUrl") || "/dashboard";

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoading(true);
    setError("");

    const res = await signIn("credentials", {
      username,
      password,
      redirect: false,
    });

    if (res?.error) {
      setError("Invalid credentials. Please try again.");
      setLoading(false);
    } else {
      // Use window.location.href for a hard-redirect. 
      // This ensures the session is fully picked up by the browser and 
      // prevents hanging issues on the first login attempt.
      window.location.href = callbackUrl;
    }
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-4 bg-[var(--background)]">
      <div className="w-full max-w-[380px]">
        <div className="flex flex-col items-center mb-8">
          <div className="h-12 w-12 rounded-xl bg-teal-600 flex items-center justify-center text-white shadow-lg shadow-teal-600/20 mb-4">
            <Terminal size={24} strokeWidth={2.5} />
          </div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white tracking-tight leading-none">TermViewer</h1>
          <p className="text-xs font-bold text-teal-600 dark:text-teal-400 mt-2 tracking-widest uppercase">Relay Plane</p>
        </div>

        <div className="surface-card p-6 sm:p-8">
          <div className="text-center mb-8">
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Welcome back</h2>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">Please enter your details to sign in.</p>
          </div>

          <form onSubmit={handleLogin} className="space-y-5">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Username</label>
              <input
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="input-field h-10"
                placeholder="Enter your username"
                autoFocus
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Password</label>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="input-field h-10"
                placeholder="••••••••"
              />
            </div>

            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/10 text-red-600 dark:text-red-400 text-xs rounded border border-red-100 dark:border-red-900/20">
                {error}
              </div>
            )}

            <button type="submit" disabled={loading} className="button-primary w-full h-10 text-sm mt-2">
              {loading ? (
                <LoaderCircle size={18} className="animate-spin" />
              ) : (
                <>Sign in <ArrowRight size={16} className="ml-2" /></>
              )}
            </button>
          </form>

          <div className="mt-8 text-center text-sm text-slate-500">
            Don&apos;t have an account?{" "}
            <Link href="/register" className="font-semibold text-teal-600 hover:text-teal-700 dark:text-teal-400 hover:underline">
              Request access
            </Link>
          </div>
        </div>

        <div className="mt-8 flex items-center justify-center gap-2 text-xs font-semibold text-slate-400">
          <Shield size={14} /> Secured via OIDC
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center">
        <LoaderCircle className="animate-spin text-teal-600" size={32} />
      </div>
    }>
      <LoginForm />
    </Suspense>
  );
}
