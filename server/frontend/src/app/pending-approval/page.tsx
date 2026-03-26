import { Clock3, MailCheck, ShieldCheck, ArrowRight } from "lucide-react";
import Link from "next/link";

import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";

type PendingApprovalSearchParams = Promise<{
  email?: string;
}>;

export default async function PendingApprovalPage({
  searchParams,
}: {
  searchParams: PendingApprovalSearchParams;
}) {
  const { email } = await searchParams;

  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content py-12">
        <div className="max-w-2xl mx-auto">
          <div className="mb-10">
            <div className="inline-flex items-center gap-1.5 rounded-full bg-[var(--warning-muted)] px-2.5 py-1 text-[11px] font-mono text-[var(--warning)] mb-4">
              <span className="inline-block h-1.5 w-1.5 rounded-full bg-[var(--warning)] animate-[tv-pulse_2s_ease-in-out_infinite]" />
              pending review
            </div>
            <h1 className="page-title">Awaiting admin approval</h1>
            <p className="section-copy mt-2">
              Your account has been created but remains locked. TermViewer enforces explicit role-gated approvals before access is granted.
            </p>
          </div>

          {email && (
            <div className="surface-inset mb-8 text-center">
              <p className="eyebrow mb-1">target email</p>
              <p className="text-sm font-mono font-medium text-[var(--text-primary)]">{email}</p>
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-3 mb-10">
            {[
              {
                label: "01 review",
                text: "Admin reviews your request in the secure console.",
                icon: ShieldCheck,
              },
              {
                label: "02 dispatch",
                text: "Approval sends a one-time activation token via email.",
                icon: MailCheck,
              },
              {
                label: "03 activate",
                text: "Click the link within 24h to unlock dashboard access.",
                icon: Clock3,
              },
            ].map((item) => (
              <div key={item.label} className="surface-card p-4">
                <item.icon size={16} className="text-[var(--accent)] mb-3" />
                <p className="eyebrow mb-1">{item.label}</p>
                <p className="text-xs text-[var(--text-secondary)] leading-relaxed">{item.text}</p>
              </div>
            ))}
          </div>

          <div className="flex flex-col gap-3 sm:flex-row items-center pt-6 border-t border-[var(--border)]">
            <Link href="/" className="button-ghost h-9 px-5 text-sm">Home</Link>
            <Link href="/login" className="button-secondary h-9 px-5 text-sm">
              Sign in (after activation) <ArrowRight size={14} />
            </Link>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
