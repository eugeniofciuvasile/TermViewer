"use client";

import { Terminal, LayoutDashboard, Settings, Menu, X, ChevronRight } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession } from "next-auth/react";
import { useState } from "react";

import { LogoutButton } from "@/components/auth-buttons";
import { ThemeToggle } from "@/components/theme-toggle";
import { cn } from "@/lib/cn";

const ICON_SIZE = 16;

export default function SiteHeader({ className }: { className?: string }) {
  const pathname = usePathname();
  const { data: session, status } = useSession();
  const [mobileOpen, setMobileOpen] = useState(false);

  const navItems = [
    { href: "/", label: "Overview", icon: Terminal },
  ];

  if (session) {
    navItems.push({ href: "/dashboard", label: "Fleet", icon: LayoutDashboard });
  }

  if (session?.isAdmin) {
    navItems.push({ href: "/admin", label: "Admin", icon: Settings });
  }

  return (
    <header className={cn("sticky top-0 z-40 w-full", className)}>
      <div className="mx-auto max-w-[72rem] px-5 sm:px-8">
        <div className="flex items-center justify-between h-16 px-5 my-3 rounded-2xl bg-[var(--surface)] border border-[var(--border)] shadow-[var(--shadow-sm)]">

          {/* ── Left: Logo + Nav ── */}
          <div className="flex items-center gap-4 md:gap-8">
            <Link href="/" className="flex items-center gap-3" onClick={() => setMobileOpen(false)}>
              <div className="h-8 w-8 rounded-[10px] bg-[var(--accent)] flex items-center justify-center shadow-[0_0_14px_var(--accent-glow)]">
                <Terminal size={ICON_SIZE} strokeWidth={2.5} className="text-[var(--accent-fg)]" />
              </div>
              <span className="text-[15px] font-semibold text-[var(--text-primary)] tracking-tight">TermViewer</span>
            </Link>

            <nav className="hidden md:flex items-center gap-1.5">
              {navItems.map((item) => {
                const isActive = pathname === item.href;
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={cn(
                      "flex items-center gap-2 px-3.5 py-2 text-sm font-medium rounded-[10px] transition-all duration-150",
                      isActive
                        ? "text-[var(--text-primary)] bg-[var(--surface-secondary)]"
                        : "text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-[var(--surface-secondary)]"
                    )}
                  >
                    <Icon size={ICON_SIZE} strokeWidth={1.75} />
                    {item.label}
                  </Link>
                );
              })}
            </nav>
          </div>

          {/* ── Right ── */}
          <div className="flex items-center gap-3">
            <ThemeToggle />

            {status === "authenticated" && session ? (
              <div className="flex items-center gap-3">
                <div className="hidden sm:flex items-center gap-2.5 px-3 py-2 rounded-[10px] bg-[var(--surface-secondary)]">
                  <div className="h-6 w-6 rounded-lg bg-[var(--accent-muted)] text-[var(--accent)] flex items-center justify-center text-[11px] font-mono font-semibold">
                    {(session.user?.name ?? session.user?.email ?? "U")[0].toUpperCase()}
                  </div>
                  <span className="text-sm font-medium text-[var(--text-secondary)]">
                    {session.user?.name ?? session.user?.email?.split("@")[0]}
                  </span>
                  {session.isAdmin && (
                    <span className="text-[10px] font-mono text-[var(--accent)] opacity-75">admin</span>
                  )}
                </div>
                <LogoutButton variant="ghost" className="button-icon text-[var(--text-muted)] hover:text-[var(--danger)] hover:bg-[var(--danger-muted)]">
                  {""}
                </LogoutButton>
              </div>
            ) : status === "loading" ? (
              <div className="h-9 w-20 animate-pulse rounded-[10px] bg-[var(--surface-secondary)]" />
            ) : (
              <Link href="/login" className="button-primary button-sm">
                Sign in <ChevronRight size={14} />
              </Link>
            )}

            <button
              className="md:hidden button-icon text-[var(--text-secondary)]"
              onClick={() => setMobileOpen(!mobileOpen)}
              aria-label="Toggle menu"
            >
              {mobileOpen ? <X size={ICON_SIZE} /> : <Menu size={ICON_SIZE} />}
            </button>
          </div>
        </div>
      </div>

      {/* ── Mobile nav ── */}
      {mobileOpen && (
        <div className="md:hidden px-5 sm:px-8 pb-2">
          <div className="mx-auto max-w-[72rem] bg-[var(--surface)] border border-[var(--border)] rounded-2xl shadow-[var(--shadow-md)] animate-slide-down overflow-hidden">
            <nav className="p-2.5 flex flex-col gap-1">
              {navItems.map((item) => {
                const isActive = pathname === item.href;
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setMobileOpen(false)}
                    className={cn(
                      "flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-xl transition-all duration-150",
                      isActive
                        ? "text-[var(--text-primary)] bg-[var(--surface-secondary)]"
                        : "text-[var(--text-secondary)] hover:bg-[var(--surface-secondary)]"
                    )}
                  >
                    <Icon size={ICON_SIZE} strokeWidth={1.75} />
                    {item.label}
                  </Link>
                );
              })}
            </nav>
          </div>
        </div>
      )}
    </header>
  );
}
