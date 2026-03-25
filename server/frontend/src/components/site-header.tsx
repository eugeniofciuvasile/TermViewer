"use client";

import { Terminal, LayoutDashboard, Settings } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession } from "next-auth/react";

import { LogoutButton } from "@/components/auth-buttons";
import { ThemeToggle } from "@/components/theme-toggle";
import { cn } from "@/lib/cn";

export default function SiteHeader({ className }: { className?: string }) {
  const pathname = usePathname();
  const { data: session, status } = useSession();

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
    <header className={cn(
      "sticky top-0 z-40 w-full bg-white dark:bg-[#0f172a] border-b-2 border-teal-600 dark:border-teal-500 shadow-sm transition-all duration-300",
      className
    )}>
      <div className="page-content py-0 h-16 flex items-center justify-between">

        {/* Left: Branding & Nav */}
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-3">
            <div className="text-teal-600 dark:text-teal-400">
              <Terminal size={24} strokeWidth={2.5} />
            </div>
            <div className="flex flex-col">
              <span className="text-base font-bold text-slate-900 dark:text-white leading-none tracking-tight">TermViewer</span>
              <span className="text-[10px] text-slate-500 dark:text-slate-400 uppercase tracking-widest mt-1 leading-none">Relay Plane</span>
            </div>
          </Link>

          <div className="h-6 w-px bg-slate-200 dark:bg-slate-700 hidden md:block" />

          <nav className="hidden md:flex items-center gap-2">
            {navItems.map((item) => {
              const isActive = pathname === item.href;
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "flex items-center gap-2 px-3 py-2 text-sm font-semibold rounded-md transition-colors",
                    isActive
                      ? "bg-slate-100 text-slate-900 dark:bg-slate-800 dark:text-white"
                      : "text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800/50 dark:hover:text-white"
                  )}
                >
                  <Icon size={16} />
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </div>

        {/* Right: Auth & Theme */}
        <div className="flex items-center gap-4">
          <ThemeToggle />

          <div className="h-6 w-px bg-slate-200 dark:bg-slate-700 hidden sm:block" />

          {status === "authenticated" && session ? (
            <div className="flex items-center gap-4">
              <div className="hidden sm:flex flex-col items-end">
                <span className="text-sm font-semibold text-slate-900 dark:text-white leading-none">
                  {session.user?.name ?? session.user?.email?.split('@')[0]}
                </span>
                <span className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  {session.isAdmin ? "Administrator" : "User"}
                </span>
              </div>
              <LogoutButton variant="ghost" className="h-9 w-9 p-0 rounded-full hover:bg-red-50 hover:text-red-600 hover:border-red-200 dark:hover:bg-red-900/20 dark:hover:text-red-400 transition-colors">
                {""}
              </LogoutButton>
            </div>
          ) : status === "loading" ? (
            <div className="h-9 w-9 animate-pulse rounded-full bg-slate-100 dark:bg-slate-800" />
          ) : (
            <Link href="/login" className="button-primary h-9">
              Sign In
            </Link>
          )}
        </div>

      </div>
    </header>
  );
}
