import Link from "next/link";
import { ArrowRight, Globe, QrCode, Shield, Sparkles, Terminal, CheckCircle2 } from "lucide-react";

import SiteHeader from "@/components/site-header";

const featureCards = [
  {
    icon: Shield,
    title: "Identity-first access",
    description:
      "Users authenticate through Keycloak, admin approvals stay role-gated, and machines never expose enrollment secrets through the share flow.",
  },
  {
    icon: QrCode,
    title: "Ephemeral QR sessions",
    description:
      "The dashboard creates short-lived share links only while a machine is online and waiting, then rotates them before expiry.",
  },
  {
    icon: Globe,
    title: "Internet-ready relay",
    description:
      "TermViewer keeps the fast LAN workflow while adding a proper public control plane for remote machine selection and relay attachment.",
  },
];

const workflowSteps = [
  "Users request access and wait in a protected admin approval queue.",
  "Approved requests trigger a one-time activation email with a 24-hour validity window.",
  "Authenticated users register machines and keep agents online through heartbeats.",
  "The mobile app scans a short-lived QR or selects an active machine from the authenticated list.",
];

export default function Home() {
  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content py-8 sm:py-12">
        <section className="grid gap-8 xl:grid-cols-[1fr_0.8fr] xl:items-start mb-12">
          <div className="surface-card p-8 sm:p-12 border-t-4 border-teal-600">
            <div className="inline-flex items-center gap-2 rounded-full border border-teal-200 bg-teal-50 px-3 py-1 text-xs font-bold text-teal-800 dark:border-teal-900/30 dark:bg-teal-900/20 dark:text-teal-300 uppercase tracking-widest mb-6">
              <Sparkles size={14} />
              Production Ready
            </div>

            <h1 className="page-title text-4xl sm:text-5xl font-black tracking-tight mb-6 leading-tight">
              Remote shell access with a cleaner security model.
            </h1>

            <p className="section-copy text-base sm:text-lg mb-8 max-w-2xl text-slate-600 dark:text-slate-400">
              TermViewer turns the terminal into a controlled product surface: approved users, machine-scoped enrollment,
              short-lived share sessions, and a relay path that works well beyond the local network.
            </p>

            <div className="flex flex-col sm:flex-row items-center gap-4">
              <Link href="/dashboard" className="button-primary h-12 px-8 text-sm w-full sm:w-auto shadow-lg shadow-teal-600/20">
                Open Dashboard
                <ArrowRight size={18} className="ml-2" />
              </Link>
              <Link href="/register" className="button-secondary h-12 px-8 text-sm w-full sm:w-auto">
                Request Access
              </Link>
            </div>

            <div className="mt-12 pt-8 border-t border-slate-100 dark:border-slate-800 grid grid-cols-1 sm:grid-cols-3 gap-6">
              {[
                { title: "OIDC + Roles", text: "Identity management & admin gates" },
                { title: "Ephemeral QR", text: "Rotating session tokens" },
                { title: "Live Presence", text: "Real-time fleet monitoring" },
              ].map((item) => (
                <div key={item.title}>
                  <p className="text-sm font-bold text-slate-900 dark:text-white mb-1">{item.title}</p>
                  <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">{item.text}</p>
                </div>
              ))}
            </div>
          </div>

          <aside className="space-y-6">
            <div className="surface-card p-8">
              <div className="flex items-center gap-4 mb-6">
                <div className="h-12 w-12 rounded-xl bg-teal-50 dark:bg-teal-900/20 text-teal-600 dark:text-teal-400 flex items-center justify-center">
                  <Terminal size={24} />
                </div>
                <div>
                  <p className="eyebrow mb-1">Operating Flow</p>
                  <h2 className="text-lg font-bold text-slate-900 dark:text-white leading-tight">Approval to Shell</h2>
                </div>
              </div>

              <div className="space-y-4">
                {workflowSteps.map((step, index) => (
                  <div key={index} className="flex gap-4">
                    <div className="flex-shrink-0 h-6 w-6 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-500 flex items-center justify-center text-xs font-bold mt-0.5">
                      {index + 1}
                    </div>
                    <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">{step}</p>
                  </div>
                ))}
              </div>
            </div>

            <div className="surface-card p-8 bg-slate-50 dark:bg-slate-900/50">
              <p className="eyebrow mb-4">Core Capabilities</p>
              <div className="space-y-3">
                {[
                  "Keycloak realm-role enforcement",
                  "Machine presence and heartbeat",
                  "Mobile OIDC and QR attachment",
                ].map((item, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <CheckCircle2 size={16} className="text-teal-500 shrink-0" />
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-300">{item}</span>
                  </div>
                ))}
              </div>
            </div>
          </aside>
        </section>

        <section className="mb-8">
          <div className="mb-8 text-center max-w-2xl mx-auto">
            <p className="eyebrow mb-2">Product Foundation</p>
            <h2 className="section-title text-2xl">A Control Plane, Not a Prototype.</h2>
            <p className="section-copy mt-3">
              Centered on surfaces that matter in production: controlled onboarding, machine visibility, audited approvals, and secure relay sessions.
            </p>
          </div>

          <div className="grid gap-6 md:grid-cols-3">
            {featureCards.map((feature, i) => {
              const Icon = feature.icon;
              return (
                <div key={i} className="surface-card p-8 group hover:-translate-y-1 transition-transform duration-300">
                  <div className="h-10 w-10 rounded-lg bg-teal-50 dark:bg-teal-900/20 text-teal-600 dark:text-teal-400 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                    <Icon size={20} strokeWidth={2.5} />
                  </div>
                  <h3 className="text-base font-bold text-slate-900 dark:text-white mb-2">{feature.title}</h3>
                  <p className="text-sm text-slate-500 dark:text-slate-400 leading-relaxed">{feature.description}</p>
                </div>
              );
            })}
          </div>
        </section>
      </main>
    </div>
  );
}
