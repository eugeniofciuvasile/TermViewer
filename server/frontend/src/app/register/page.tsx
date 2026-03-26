"use client";

import axios from "axios";
import { ArrowRight, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";

export default function RegisterPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleRegister = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      await axios.post(`${process.env.NEXT_PUBLIC_BACKEND_URL}/api/register`, {
        username,
        email,
        password,
      });
      router.push(`/pending-approval?email=${encodeURIComponent(email)}`);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        setError(err.response?.data?.error || "Registration failed");
      } else {
        setError("Registration failed");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content py-12">
        <div className="max-w-3xl mx-auto">
          <div className="mb-10">
            <p className="eyebrow mb-2">onboarding</p>
            <h1 className="page-title">Request access</h1>
            <p className="section-copy mt-2">
              Create your identity. All requests are manually reviewed by administrators.
            </p>
          </div>

          <div className="grid gap-8 md:grid-cols-[1fr_0.8fr] items-start">
            <div className="surface-card overflow-hidden">
              <h2 className="section-title mb-6">Account details</h2>
              <form onSubmit={handleRegister} className="space-y-6">
                <div>
                  <label className="input-label">Username</label>
                  <input
                    type="text"
                    required
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    className="input-field"
                    placeholder="johndoe"
                  />
                </div>

                <div>
                  <label className="input-label">Email</label>
                  <input
                    type="email"
                    required
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    className="input-field"
                    placeholder="john@example.com"
                  />
                </div>

                <div>
                  <label className="input-label">Password</label>
                  <input
                    type="password"
                    required
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    className="input-field"
                    placeholder="Strong password"
                  />
                </div>

                {error && (
                  <div className="alert alert-danger animate-slide-up">
                    <span className="text-sm">{error}</span>
                  </div>
                )}

                <div className="pt-4 border-t border-[var(--border)]">
                  <button type="submit" disabled={loading} className="button-primary w-full">
                    {loading ? "Submitting..." : "Submit Request"}
                    {!loading && <ArrowRight size={14} />}
                  </button>
                </div>
              </form>
            </div>

            <div className="space-y-6">
              <div className="surface-panel">
                <p className="eyebrow mb-4">process</p>
                <div className="space-y-4">
                  {[
                    "Admin reviews every request before login is enabled.",
                    "Approval sends a one-time activation email (24h lifetime).",
                    "Machine credentials are created in the dashboard after auth.",
                  ].map((item, i) => (
                    <div key={i} className="flex gap-3">
                      <ShieldCheck size={16} className="text-[var(--accent)] shrink-0 mt-0.5" />
                      <p className="text-sm text-[var(--text-secondary)] leading-relaxed">{item}</p>
                    </div>
                  ))}
                </div>
              </div>

              <div className="text-center text-sm text-[var(--text-muted)]">
                Already activated?{" "}
                <Link href="/login" className="font-medium text-[var(--accent)] hover:underline">
                  Sign in
                </Link>
              </div>
            </div>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
