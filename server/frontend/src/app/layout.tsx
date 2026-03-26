import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import AuthProvider from "@/providers/auth-provider";
import SessionGuard from "@/providers/session-guard";
import { ThemeProvider } from "@/providers/theme-provider";

import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "TermViewer",
    template: "%s · TermViewer",
  },
  description:
    "Secure terminal streaming — LAN discovery, relay access, fleet management, and session sharing.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}>
      <body className="min-h-full">
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem disableTransitionOnChange>
          <AuthProvider>
            <SessionGuard>{children}</SessionGuard>
          </AuthProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
