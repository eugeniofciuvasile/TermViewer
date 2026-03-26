"use client";

import { signIn } from "next-auth/react";
import { useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";
import { ArrowRight, LoaderCircle, Terminal, ArrowLeft } from "lucide-react";
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
      window.location.href = callbackUrl;
    }
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-6 bg-[var(--canvas)]">
      <div className="w-full max-w-[400px]">
        <Link href="/" className="inline-flex items-center gap-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors mb-12">
          <ArrowLeft size={14} />
          Back
        </Link>

        <div className="flex items-center gap-3 mb-12">
          <div className="h-8 w-8 rounded-[10px] bg-[var(--accent)] flex items-center justify-center">
            <Terminal size={16} strokeWidth={2.5} className="text-[var(--accent-fg)]" />
          </div>
          <span className="text-base font-semibold text-[var(--text-primary)] tracking-tight">TermViewer</span>
        </div>

        <div className="mb-10">
          <h1 className="text-2xl font-semibold text-[var(--text-primary)] tracking-tight">Sign in</h1>
          <p className="text-sm text-[var(--text-secondary)] mt-2">Access the terminal control plane.</p>
        </div>

        <form onSubmit={handleLogin} className="space-y-6">
          <div>
            <label className="input-label">Username</label>
            <input
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="input-field"
              placeholder="username"
              autoFocus
            />
          </div>

          <div>
            <label className="input-label">Password</label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input-field"
              placeholder="••••••••"
            />
          </div>

          {error && (
            <div className="alert alert-danger animate-slide-up">
              <span className="text-sm">{error}</span>
            </div>
          )}

          <button type="submit" disabled={loading} className="button-primary w-full">
            {loading ? (
              <LoaderCircle size={16} className="animate-spin" />
            ) : (
              <>Sign in <ArrowRight size={14} /></>
            )}
          </button>
        </form>

        <div className="mt-10 pt-6 border-t border-[var(--border)] text-center text-sm text-[var(--text-muted)]">
          No account?{" "}
          <Link href="/register" className="font-medium text-[var(--accent)] hover:underline">
            Request access
          </Link>
        </div>

        <div className="mt-10 flex items-center justify-center gap-2 text-xs font-mono text-[var(--text-muted)]">
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-[var(--success)]" />
          secured via keycloak oidc
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-[var(--canvas)]">
        <LoaderCircle className="animate-spin text-[var(--accent)]" size={24} />
      </div>
    }>
      <LoginForm />
    </Suspense>
  );
}
