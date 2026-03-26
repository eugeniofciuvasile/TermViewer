import { CheckCircle2, ShieldAlert, ArrowRight } from "lucide-react";
import Link from "next/link";

import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";
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

      <main className="page-content py-16 flex justify-center">
        <div className="max-w-lg w-full text-center">
          <div className={cn(
            "mx-auto flex h-14 w-14 items-center justify-center rounded-xl mb-6",
            isSuccess ? "bg-[var(--success-muted)] text-[var(--success)]" : "bg-[var(--danger-muted)] text-[var(--danger)]"
          )}>
            {isSuccess ? <CheckCircle2 size={28} /> : <ShieldAlert size={28} />}
          </div>
          
          <h1 className="page-title">{isSuccess ? "Account activated" : "Activation failed"}</h1>
          <p className="section-copy mt-3 max-w-sm mx-auto">{result.message}</p>

          <div className="mt-10 grid gap-3 sm:grid-cols-2 text-left">
            <div className="surface-panel p-4">
              <p className="eyebrow mb-1.5">next steps</p>
              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                {isSuccess 
                  ? "Sign in to register machines and manage relay sessions."
                  : "Contact an administrator to restart the approval cycle."}
              </p>
            </div>
            <div className="surface-panel p-4">
              <p className="eyebrow mb-1.5">security</p>
              <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
                Activation is mandatory after admin approval. Only activated accounts can use the OIDC flow.
              </p>
            </div>
          </div>

          <div className="mt-8 flex flex-col gap-2 sm:flex-row justify-center">
            {isSuccess && (
              <Link href="/login" className="button-primary h-9 px-6 text-sm">
                Sign in <ArrowRight size={14} />
              </Link>
            )}
            <Link href="/" className="button-ghost h-9 px-5 text-sm">
              Home
            </Link>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
