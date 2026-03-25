"use client";

import { useTheme } from "next-themes";
import { Moon, Sun } from "lucide-react";
import { useState } from "react";

export function ThemeToggle() {
  const [mounted] = useState(() => {
    if (typeof window !== "undefined") return true;
    return false;
  });
  const { theme, setTheme } = useTheme();

  if (!mounted) {
    return <div className="w-8 h-8 rounded-full border border-slate-200 dark:border-slate-800" />;
  }

  return (
    <button
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      className="flex h-8 w-8 items-center justify-center rounded-full border border-slate-200 text-slate-600 transition-colors hover:bg-slate-100 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800"
      aria-label="Toggle theme"
    >
      {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
    </button>
  );
}
