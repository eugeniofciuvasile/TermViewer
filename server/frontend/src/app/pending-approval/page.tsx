import { Clock3, MailCheck, ShieldCheck, ArrowRight } from "lucide-react";
import Link from "next/link";

import SiteHeader from "@/components/site-header";

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

      <main className="page-content py-10">
        <div className="surface-card mx-auto max-w-4xl p-8 sm:p-12 border-t-4 border-orange-500">
          <div className="text-center mb-10">
            <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-2xl bg-orange-50 text-orange-600 dark:bg-orange-900/20 dark:text-orange-400 mb-6 shadow-sm">
              <MailCheck size={40} />
            </div>
            <p className="eyebrow mb-2">Request Received</p>
            <h1 className="page-title">Pending Administrator Approval</h1>
            <p className="section-copy mt-4 max-w-2xl mx-auto text-base">
              Your account has been successfully created but remains locked. TermViewer enforces explicit role-gated approvals before any workspace access is granted.
            </p>
          </div>

          {email && (
            <div className="max-w-md mx-auto mb-10 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-lg p-4 text-center">
              <p className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-1">Target Email</p>
              <p className="text-sm font-semibold text-slate-900 dark:text-white">{email}</p>
            </div>
          )}

          <div className="grid gap-6 md:grid-cols-3 mb-10">
            {[
              {
                title: "1. Admin Review",
                text: "A privileged administrator reviews your request in the secure console.",
                icon: ShieldCheck,
              },
              {
                title: "2. Secure Dispatch",
                text: "Upon approval, a one-time activation token is emailed to you.",
                icon: MailCheck,
              },
              {
                title: "3. Access Granted",
                text: "Activate the link within 24h to unlock full dashboard access.",
                icon: Clock3,
              },
            ].map((item) => (
              <div key={item.title} className="surface-panel p-6 text-center">
                <item.icon size={24} className="mx-auto text-teal-600 mb-4 opacity-80" />
                <p className="text-sm font-bold text-slate-900 dark:text-white mb-2">{item.title}</p>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">{item.text}</p>
              </div>
            ))}
          </div>

          <div className="flex flex-col gap-4 sm:flex-row justify-center items-center pt-8 border-t border-slate-100 dark:border-slate-800">
            <Link href="/" className="button-ghost h-10 px-8">
              Return to Home
            </Link>
            <Link href="/login" className="button-secondary h-10 px-8">
              Sign In (After Activation) <ArrowRight size={16} className="ml-2" />
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
}
