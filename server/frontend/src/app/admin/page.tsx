import { revalidatePath } from "next/cache";
import { CheckCircle2, ShieldAlert, UserRoundCheck, ArrowRight, ShieldCheck, RefreshCw, Zap, Clock, AlertTriangle } from "lucide-react";
import { getServerSession } from "next-auth";
import Link from "next/link";
import { redirect } from "next/navigation";

import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";
import { authOptions, sessionHasAdminRole } from "@/lib/auth";
import api from "@/lib/api";

interface PendingUser {
  id: number;
  username: string;
  email: string;
  is_approved: boolean;
  is_activated: boolean;
  created_at: string;
}

async function fetchPendingUsers(accessToken: string): Promise<PendingUser[]> {
  try {
    const response = await api.get<PendingUser[]>("/api/admin/pending-users", {
      headers: { Authorization: `Bearer ${accessToken}` }
    });
    return response.data;
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to load pending users";
    throw new Error(message);
  }
}

async function fetchApprovedUsers(accessToken: string): Promise<PendingUser[]> {
  try {
    const response = await api.get<PendingUser[]>("/api/admin/approved-users", {
      headers: { Authorization: `Bearer ${accessToken}` }
    });
    return response.data;
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to load approved users";
    throw new Error(message);
  }
}

async function approvePendingUser(formData: FormData) {
  "use server";
  const session = await getServerSession(authOptions);
  if (!session?.accessToken) redirect("/login?callbackUrl=/admin");
  if (!sessionHasAdminRole(session)) redirect("/dashboard");
  
  const userId = formData.get("userId");
  if (!userId || typeof userId !== "string") throw new Error("Invalid user id.");
  
  try {
    await api.post(`/api/admin/approve/${userId}`, {}, {
      headers: { Authorization: `Bearer ${session.accessToken}` }
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to approve user";
    throw new Error(message);
  }
  
  revalidatePath("/admin");
}

async function forceActivateUser(formData: FormData) {
  "use server";
  const session = await getServerSession(authOptions);
  if (!session?.accessToken) redirect("/login?callbackUrl=/admin");
  if (!sessionHasAdminRole(session)) redirect("/dashboard");

  const userId = formData.get("userId");
  if (!userId || typeof userId !== "string") throw new Error("Invalid user id.");

  try {
    await api.post(`/api/admin/force-activate/${userId}`, {}, {
      headers: { Authorization: `Bearer ${session.accessToken}` }
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to force-activate user";
    throw new Error(message);
  }

  revalidatePath("/admin");
}

async function resendActivationEmail(formData: FormData) {
  "use server";
  const session = await getServerSession(authOptions);
  if (!session?.accessToken) redirect("/login?callbackUrl=/admin");
  if (!sessionHasAdminRole(session)) redirect("/dashboard");

  const userId = formData.get("userId");
  if (!userId || typeof userId !== "string") throw new Error("Invalid user id.");

  try {
    await api.post(`/api/admin/resend-activation/${userId}`, {}, {
      headers: { Authorization: `Bearer ${session.accessToken}` }
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "Failed to resend activation email";
    throw new Error(message);
  }

  revalidatePath("/admin");
}

export default async function AdminPage() {
  const session = await getServerSession(authOptions);
  if (!session?.accessToken) redirect("/login?callbackUrl=/admin");

  if (!sessionHasAdminRole(session)) {
    return (
      <div className="page-shell">
        <SiteHeader />
        <div className="page-content py-20 flex justify-center">
          <div className="max-w-sm w-full text-center">
            <ShieldAlert size={32} className="mx-auto text-[var(--danger)] mb-4" />
            <h1 className="section-title">Access denied</h1>
            <p className="section-copy mt-2">You lack the administrative roles required for this console.</p>
            <Link href="/dashboard" className="button-ghost w-full mt-6">Back to fleet</Link>
          </div>
        </div>
      </div>
    );
  }

  let pendingUsers: PendingUser[] = [];
  let approvedUsers: PendingUser[] = [];
  let error: string | null = null;
  try {
    [pendingUsers, approvedUsers] = await Promise.all([
      fetchPendingUsers(session.accessToken),
      fetchApprovedUsers(session.accessToken),
    ]);
  } catch (e) {
    error = e instanceof Error ? e.message : "Load failed";
  }

  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content">
        <div className="mb-8">
          <p className="eyebrow mb-2">admin</p>
          <h1 className="page-title">Approval queue</h1>
          <p className="section-copy">Review and manage workspace access requests.</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-8">
          {[
            { label: "Pending", value: pendingUsers.length, icon: UserRoundCheck },
            { label: "Awaiting activation", value: approvedUsers.length, icon: Clock },
            { label: "Policy", value: "Role-locked", icon: ShieldAlert },
          ].map((card) => (
            <div key={card.label} className="surface-card flex items-center gap-3 py-3 px-4">
              <card.icon size={16} className="text-[var(--text-muted)]" />
              <div>
                <p className="eyebrow">{card.label}</p>
                <p className="text-lg font-semibold text-[var(--text-primary)] leading-tight mt-0.5">{card.value}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="grid gap-8 lg:grid-cols-[1fr_280px] items-start">
          <div className="space-y-8">
            {/* Pending Approval */}
            <div className="surface-card p-0 overflow-hidden">
              <div className="px-4 py-3 border-b border-[var(--border)]">
                <h2 className="text-[13px] font-medium text-[var(--text-primary)]">Pending approval</h2>
              </div>
              <div className="overflow-x-auto">
                <table className="data-table min-w-[480px]">
                  <thead>
                    <tr>
                      <th>Identity</th>
                      <th>Requested</th>
                      <th className="text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {error ? (
                      <tr>
                        <td colSpan={3} className="text-center py-8">
                          <span className="p-tag p-tag-danger">{error}</span>
                        </td>
                      </tr>
                    ) : pendingUsers.length === 0 ? (
                      <tr>
                        <td colSpan={3} className="text-center py-10 text-[var(--text-muted)] text-sm">
                          <ShieldCheck size={24} className="mx-auto text-[var(--accent)] mb-2 opacity-40" />
                          No pending requests.
                        </td>
                      </tr>
                    ) : (
                      pendingUsers.map((user) => (
                        <tr key={user.id}>
                          <td>
                            <div>
                              <p className="text-[13px] font-medium text-[var(--text-primary)]">{user.username}</p>
                              <p className="text-xs font-mono text-[var(--text-muted)] mt-0.5">{user.email}</p>
                            </div>
                          </td>
                          <td className="text-[13px] text-[var(--text-secondary)]">
                            {new Date(user.created_at).toLocaleDateString()}
                          </td>
                          <td className="text-right">
                            <form action={approvePendingUser}>
                              <input type="hidden" name="userId" value={user.id} />
                              <button type="submit" className="button-primary h-7 px-3 text-xs">
                                Approve <ArrowRight size={12} />
                              </button>
                            </form>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Pending Activation */}
            <div className="surface-card p-0 overflow-hidden">
              <div className="px-4 py-3 border-b border-[var(--border)] flex items-center gap-2">
                <AlertTriangle size={13} className="text-[var(--warning)]" />
                <h2 className="text-[13px] font-medium text-[var(--text-primary)]">Pending activation</h2>
                <span className="text-[11px] text-[var(--text-muted)] ml-1">approved but not yet activated</span>
              </div>
              <div className="overflow-x-auto">
                <table className="data-table min-w-[480px]">
                  <thead>
                    <tr>
                      <th>Identity</th>
                      <th>Approved</th>
                      <th className="text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {approvedUsers.length === 0 ? (
                      <tr>
                        <td colSpan={3} className="text-center py-10 text-[var(--text-muted)] text-sm">
                          <CheckCircle2 size={24} className="mx-auto text-[var(--accent)] mb-2 opacity-40" />
                          All users activated.
                        </td>
                      </tr>
                    ) : (
                      approvedUsers.map((user) => (
                        <tr key={user.id}>
                          <td>
                            <div>
                              <p className="text-[13px] font-medium text-[var(--text-primary)]">{user.username}</p>
                              <p className="text-xs font-mono text-[var(--text-muted)] mt-0.5">{user.email}</p>
                            </div>
                          </td>
                          <td className="text-[13px] text-[var(--text-secondary)]">
                            {new Date(user.created_at).toLocaleDateString()}
                          </td>
                          <td className="text-right">
                            <div className="flex items-center justify-end gap-1.5">
                              <form action={resendActivationEmail}>
                                <input type="hidden" name="userId" value={user.id} />
                                <button type="submit" className="button-ghost h-7 px-2.5 text-xs">
                                  <RefreshCw size={12} /> Resend
                                </button>
                              </form>
                              <form action={forceActivateUser}>
                                <input type="hidden" name="userId" value={user.id} />
                                <button type="submit" className="button-primary h-7 px-2.5 text-xs" style={{ background: 'var(--warning)', borderColor: 'var(--warning)' }}>
                                  <Zap size={12} /> Force
                                </button>
                              </form>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* Sidebar */}
          <aside className="surface-card">
            <h2 className="section-title mb-4">Security policy</h2>
            <div className="space-y-4">
              {[
                { title: "Verification", text: "Admin role strictly required to view or approve." },
                { title: "Activation", text: "Approvals dispatch a one-time activation email." },
                { title: "Expiry", text: "Activation tokens expire after 24 hours." },
                { title: "Force activate", text: "Bypasses email — use when delivery fails." },
              ].map((item, i) => (
                <div key={i} className="flex gap-2.5">
                  <CheckCircle2 size={14} className="text-[var(--accent)] shrink-0 mt-0.5" />
                  <div>
                    <p className="text-xs font-medium text-[var(--text-primary)]">{item.title}</p>
                    <p className="text-[11px] text-[var(--text-muted)] mt-0.5 leading-relaxed">{item.text}</p>
                  </div>
                </div>
              ))}
            </div>
          </aside>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
