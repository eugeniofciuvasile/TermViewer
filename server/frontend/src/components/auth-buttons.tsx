"use client";

import { LoaderCircle, LogIn, LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import { signOut } from "next-auth/react";
import { type ReactNode, useState } from "react";

import { cn } from "@/lib/cn";

type ButtonVariant = "primary" | "secondary" | "ghost";

const variantClassNames: Record<ButtonVariant, string> = {
  primary: "button-primary",
  secondary: "button-secondary",
  ghost: "button-ghost",
};

interface SignInButtonProps {
  callbackUrl?: string;
  className?: string;
  forcePrompt?: boolean;
  children?: ReactNode;
  variant?: ButtonVariant;
}

export function SignInButton({
  callbackUrl = "/dashboard",
  className,
  children,
  variant = "primary",
}: SignInButtonProps) {
  const router = useRouter();

  return (
    <button
      type="button"
      onClick={() => router.push(`/login?callbackUrl=${encodeURIComponent(callbackUrl)}`)}
      className={cn(variantClassNames[variant], className)}
    >
      <LogIn size={16} />
      {children ?? "Sign in"}
    </button>
  );
}

interface LogoutButtonProps {
  callbackUrl?: string;
  className?: string;
  children?: ReactNode;
  logoutUrl?: string;
  variant?: ButtonVariant;
}

export function LogoutButton({
  callbackUrl = "/",
  className,
  children,
  logoutUrl,
  variant = "ghost",
}: LogoutButtonProps) {
  const [loading, setLoading] = useState(false);

  const handleClick = async () => {
    setLoading(true);

    try {
      if (logoutUrl) {
        await signOut({ redirect: false, callbackUrl });
        window.location.assign(logoutUrl);
        return;
      }

      await signOut({ callbackUrl });
    } catch (error) {
      console.error("Logout failed", error);
      setLoading(false);
    }
  };

  return (
    <button
      type="button"
      onClick={() => void handleClick()}
      disabled={loading}
      className={cn(variantClassNames[variant], className)}
    >
      {loading ? <LoaderCircle size={16} className="animate-spin" /> : <LogOut size={16} />}
      {children ?? "Logout"}
    </button>
  );
}
