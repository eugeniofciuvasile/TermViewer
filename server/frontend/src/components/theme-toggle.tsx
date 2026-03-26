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
    return <div className="w-9 h-9 rounded-[10px] bg-[var(--surface-secondary)]" />;
  }

  return (
    <button
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      className="button-icon"
      aria-label="Toggle theme"
    >
      {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
    </button>
  );
}
