import { Terminal, Github } from "lucide-react";
import Link from "next/link";

export default function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="mx-auto max-w-[72rem] px-5 sm:px-8">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-2.5">
            <Terminal size={14} className="text-[var(--text-muted)]" />
            <span className="text-xs font-mono text-[var(--text-muted)]">termviewer v0.1.0</span>
          </div>

          <nav className="flex items-center gap-6">
            <Link href="/" className="text-xs text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors">
              Overview
            </Link>
            <Link href="/login" className="text-xs text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors">
              Sign in
            </Link>
            <a
              href="https://github.com/eugeniofciuvasile/TermViewer"
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors flex items-center gap-1.5"
            >
              <Github size={13} />
              GitHub
            </a>
          </nav>
        </div>
      </div>
    </footer>
  );
}
