"use client";

import axios from "axios";
import { ArrowRight, MailPlus, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import SiteHeader from "@/components/site-header";

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

      <main className="page-content py-8">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-10">
            <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-teal-50 text-teal-600 dark:bg-teal-900/20 dark:text-teal-400 mb-6">
              <MailPlus size={32} />
            </div>
            <h1 className="page-title">Request Workspace Access</h1>
            <p className="section-copy mt-2 max-w-xl mx-auto">
              Create your identity to access the public relay control plane. All requests are manually reviewed by administrators.
            </p>
          </div>

          <div className="grid gap-8 md:grid-cols-2 items-start">
            <div className="surface-card p-8">
              <h2 className="section-title mb-6">Account Details</h2>
              <form onSubmit={handleRegister} className="space-y-5">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Username</label>
                  <input
                    type="text"
                    required
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    className="input-field h-10"
                    placeholder="johndoe"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Email Address</label>
                  <input
                    type="email"
                    required
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    className="input-field h-10"
                    placeholder="john@example.com"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">Password</label>
                  <input
                    type="password"
                    required
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    className="input-field h-10"
                    placeholder="Choose a strong password"
                  />
                </div>

                {error && (
                  <div className="p-3 bg-red-50 dark:bg-red-900/10 text-red-600 dark:text-red-400 text-xs rounded border border-red-100 dark:border-red-900/20">
                    {error}
                  </div>
                )}

                <div className="pt-4 border-t border-slate-100 dark:border-slate-800">
                  <button type="submit" disabled={loading} className="button-primary w-full h-10 text-sm">
                    {loading ? "Submitting request..." : "Submit Request"}
                    {!loading && <ArrowRight size={16} className="ml-2" />}
                  </button>
                </div>
              </form>
            </div>

            <div className="space-y-6">
              <div className="surface-panel p-6">
                <p className="eyebrow mb-4">Onboarding Process</p>
                <div className="space-y-4">
                  {[
                    "An administrator reviews every request before login is enabled.",
                    "Approval sends a separate activation email with a 24-hour lifetime.",
                    "Machine credentials are created only inside the dashboard after authentication.",
                  ].map((item, i) => (
                    <div key={i} className="flex gap-3">
                      <ShieldCheck size={18} className="text-teal-600 shrink-0" />
                      <p className="text-sm text-slate-600 dark:text-slate-400">{item}</p>
                    </div>
                  ))}
                </div>
              </div>

              <div className="text-center text-sm text-slate-500">
                Already activated?{" "}
                <Link href="/login" className="font-semibold text-teal-600 hover:text-teal-700 dark:text-teal-400 hover:underline">
                  Sign in to dashboard
                </Link>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
