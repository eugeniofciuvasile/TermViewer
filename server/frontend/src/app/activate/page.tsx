import { CheckCircle2, ShieldAlert, ArrowRight } from "lucide-react";
import Link from "next/link";

import SiteHeader from "@/components/site-header";
import { cn } from "@/lib/cn";

type ActivationState = "success" | "error";

type ActivationSearchParams = Promise<{
  token?: string;
}>;

const backendUrl = process.env.NEXT_PUBLIC_BACKEND_URL;

async function activateAccount(token: string): Promise<{ state: ActivationState; message: string }> {
  if (!backendUrl) {
    return {
      state: "error",
      message: "NEXT_PUBLIC_BACKEND_URL is not configured.",
    };
  }

  try {
    const response = await fetch(`${backendUrl}/api/activate?token=${encodeURIComponent(token)}`, {
      cache: "no-store",
    });
    const data = (await response.json()) as { message?: string; error?: string };

    if (!response.ok) {
      return {
        state: "error",
        message: data.error ?? "Activation failed.",
      };
    }

    return {
      state: "success",
      message: data.message ?? "Account activated successfully.",
    };
  } catch {
    return {
      state: "error",
      message: "Activation failed because the server could not be reached.",
    };
  }
}

export default async function ActivatePage({
  searchParams,
}: {
  searchParams: ActivationSearchParams;
}) {
  const { token } = await searchParams;
  const result = token
    ? await activateAccount(token)
    : {
        state: "error" as const,
        message: "This activation link is missing its token.",
      };

  const isSuccess = result.state === "success";

  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content py-12 flex justify-center">
        <div className="surface-card max-w-2xl w-full p-8 sm:p-10 border-t-4 text-center" style={{ borderTopColor: isSuccess ? 'var(--primary)' : '#ef4444' }}>
          <div className={cn(
            "mx-auto flex h-20 w-20 items-center justify-center rounded-2xl mb-8 shadow-sm",
            isSuccess ? "bg-teal-50 text-teal-600 dark:bg-teal-900/20 dark:text-teal-400" : "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400"
          )}>
            {isSuccess ? <CheckCircle2 size={40} /> : <ShieldAlert size={40} />}
          </div>
          
          <h1 className="page-title">{isSuccess ? "Account Activated" : "Activation Failed"}</h1>
          <p className="section-copy mt-4 max-w-lg mx-auto text-base">{result.message}</p>

          <div className="mt-10 grid gap-4 md:grid-cols-2 text-left">
            <div className="surface-panel p-5">
              <p className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider mb-2">Next Steps</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                {isSuccess 
                  ? "You can now sign in to the dashboard to register machines and manage relay sessions."
                  : "If the link expired or was already used, ask an administrator to restart the approval and activation cycle."}
              </p>
            </div>
            <div className="surface-panel p-5">
              <p className="text-xs font-bold text-slate-900 dark:text-white uppercase tracking-wider mb-2">Security Policy</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Activation is a mandatory step after administrative approval. Only activated accounts can utilize the OIDC flow.
              </p>
            </div>
          </div>

          <div className="mt-10 flex flex-col gap-3 sm:flex-row justify-center pt-8 border-t border-slate-100 dark:border-slate-800">
            {isSuccess && (
              <Link href="/login" className="button-primary h-10 px-8">
                Sign In Now <ArrowRight size={16} className="ml-2" />
              </Link>
            )}
            <Link href="/" className="button-ghost h-10 px-6">
              Return Home
            </Link>
          </div>
        </div>
      </main>
    </div>
  );
}
