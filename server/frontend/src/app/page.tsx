import Link from "next/link";
import { ArrowRight, Globe, QrCode, Shield, Terminal, Wifi, FileText, MonitorSmartphone, ChevronRight, Github, UserCheck, MailCheck, Server, Smartphone } from "lucide-react";

import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";

const features = [
  {
    icon: Shield,
    title: "Identity-first access",
    description: "OIDC via Keycloak, admin approval gates, role-based controls. Machine secrets never leak through the share flow.",
  },
  {
    icon: QrCode,
    title: "Ephemeral QR sessions",
    description: "Short-lived share tokens, auto-rotated before expiry. Connect by scanning — no credentials exchanged.",
  },
  {
    icon: Globe,
    title: "Internet relay",
    description: "WebSocket relay with TLS. Keep fast LAN discovery while adding a control plane for remote fleet access.",
  },
  {
    icon: Wifi,
    title: "Zero-config LAN",
    description: "The agent broadcasts via mDNS. Open the app on the same network — machines appear automatically.",
  },
  {
    icon: FileText,
    title: "Audit & recording",
    description: "Asciinema session recording. The public server logs every action with user, IP, and timestamp.",
  },
  {
    icon: MonitorSmartphone,
    title: "Mobile terminal",
    description: "File transfer, clipboard sync, system HUD, custom macros, pan & zoom for complex TUIs.",
  },
];

const steps = [
  { icon: UserCheck, label: "Request", description: "User submits access request to admin approval queue" },
  { icon: MailCheck, label: "Approve", description: "Admin reviews and triggers one-time activation email" },
  { icon: Server, label: "Enroll", description: "User registers machines, agent reports via heartbeat" },
  { icon: Smartphone, label: "Connect", description: "Mobile app scans QR or selects machine from fleet" },
];

export default function Home() {
  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content py-12 sm:py-20">
        {/* Hero */}
        <section className="mb-24">
          <div className="max-w-2xl">
            <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--surface-secondary)] px-3 py-1.5 text-xs font-mono text-[var(--text-secondary)] mb-8">
              <span className="inline-block h-1.5 w-1.5 rounded-full bg-[var(--success)]" />
              open source · AGPLv3
            </div>

            <h1 className="text-3xl sm:text-4xl lg:text-[2.75rem] font-semibold tracking-tight text-[var(--text-primary)] mb-6 leading-[1.15]">
              Secure terminal access<br />
              <span className="text-[var(--accent)]">from anywhere.</span>
            </h1>

            <p className="text-[15px] text-[var(--text-secondary)] mb-10 max-w-lg leading-relaxed">
              Stream terminal sessions over LAN or internet. Fleet management, OIDC auth, ephemeral QR sharing, and session recording — deployed with a single compose file.
            </p>

            <div className="flex flex-wrap items-center gap-3 mb-16">
              <Link href="/dashboard" className="button-primary h-9 px-5 text-sm">
                Open Dashboard
                <ArrowRight size={14} />
              </Link>
              <Link href="/register" className="button-secondary h-9 px-5 text-sm">
                Request Access
              </Link>
              <a
                href="https://github.com/eugeniofciuvasile/TermViewer"
                target="_blank"
                rel="noopener noreferrer"
                className="button-ghost h-9 px-4 text-sm"
              >
                <Github size={14} />
                Star on GitHub
              </a>
            </div>

            {/* Terminal-style info block */}
            <div className="surface-card p-0 overflow-hidden font-mono text-xs">
              <div className="flex items-center gap-2.5 px-5 py-3 border-b border-[var(--border)] bg-[var(--surface-secondary)]">
                <span className="h-2.5 w-2.5 rounded-full bg-[var(--danger)] opacity-60" />
                <span className="h-2.5 w-2.5 rounded-full bg-[var(--warning)] opacity-60" />
                <span className="h-2.5 w-2.5 rounded-full bg-[var(--success)] opacity-60" />
                <span className="ml-2 text-[var(--text-muted)]">overview</span>
              </div>
              <div className="p-5 space-y-2 text-[var(--text-secondary)]">
                <div><span className="text-[var(--accent)]">auth</span>     keycloak OIDC · realm-role enforcement</div>
                <div><span className="text-[var(--accent)]">fleet</span>    machine enrollment · heartbeat presence</div>
                <div><span className="text-[var(--accent)]">share</span>    ephemeral QR tokens · auto-rotation</div>
                <div><span className="text-[var(--accent)]">tenant</span>   row-level isolation · per-user machines</div>
              </div>
            </div>
          </div>
        </section>

        {/* Workflow */}
        <section className="mb-24">
          <div className="mb-10">
            <p className="eyebrow mb-2">workflow</p>
            <h2 className="text-xl font-semibold text-[var(--text-primary)] tracking-tight">From approval to shell</h2>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {steps.map((step, i) => {
              const Icon = step.icon;
              return (
                <div key={i} className="surface-card surface-card-interactive p-6 relative">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="h-9 w-9 rounded-xl bg-[var(--accent-muted)] flex items-center justify-center">
                      <Icon size={18} strokeWidth={1.75} className="text-[var(--accent)]" />
                    </div>
                    <div>
                      <span className="font-mono text-[10px] text-[var(--text-muted)] font-medium">{String(i + 1).padStart(2, "0")}</span>
                      <p className="text-sm font-semibold text-[var(--text-primary)] leading-tight">{step.label}</p>
                    </div>
                  </div>
                  <p className="text-[13px] text-[var(--text-secondary)] leading-relaxed">{step.description}</p>
                </div>
              );
            })}
          </div>
        </section>

        {/* Features */}
        <section className="mb-16">
          <div className="mb-10">
            <p className="eyebrow mb-2">capabilities</p>
            <h2 className="text-xl font-semibold text-[var(--text-primary)] tracking-tight">Not a prototype</h2>
            <p className="section-copy mt-2 max-w-lg">
              Production controls: approved onboarding, fleet visibility, audited sessions, secure relay.
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature, i) => {
              const Icon = feature.icon;
              return (
                <div key={i} className="surface-card surface-card-interactive p-6 group">
                  <div className="h-9 w-9 rounded-xl bg-[var(--accent-muted)] flex items-center justify-center mb-5 group-hover:scale-105 transition-transform duration-150">
                    <Icon size={18} strokeWidth={1.75} className="text-[var(--accent)]" />
                  </div>
                  <h3 className="text-sm font-semibold text-[var(--text-primary)] mb-2">{feature.title}</h3>
                  <p className="text-[13px] text-[var(--text-secondary)] leading-relaxed">{feature.description}</p>
                </div>
              );
            })}
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  );
}
