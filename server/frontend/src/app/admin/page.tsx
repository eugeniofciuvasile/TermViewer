import { revalidatePath } from "next/cache";
import { CheckCircle2, ShieldAlert, UserRoundCheck, ArrowRight, ShieldCheck, RefreshCw, Zap, Clock, AlertTriangle } from "lucide-react";
import { getServerSession } from "next-auth";
import Link from "next/link";
import { redirect } from "next/navigation";

import SiteHeader from "@/components/site-header";
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
          <div className="surface-card max-w-sm w-full text-center border-t-4 border-red-600">
            <ShieldAlert size={48} className="mx-auto text-red-500 mb-6" />
            <h1 className="section-title">Privileged Access</h1>
            <p className="section-copy mt-2">You lack the administrative roles required for this console.</p>
            <Link href="/dashboard" className="button-ghost w-full mt-8">Back to Safety</Link>
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
        <div className="mb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="eyebrow mb-1">Gated Access</p>
            <h1 className="page-title">Approval Queue</h1>
            <p className="section-copy">Review and manage workspace access requests.</p>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8">
          {[
            { label: "Pending Requests", value: pendingUsers.length, icon: UserRoundCheck },
            { label: "Awaiting Activation", value: approvedUsers.length, icon: Clock },
            { label: "Approval Policy", value: "Role Locked", icon: ShieldAlert },
          ].map((card) => (
            <div key={card.label} className="surface-card flex items-center gap-4">
              <div className="h-12 w-12 rounded-lg bg-slate-100 dark:bg-slate-800 text-slate-500 flex items-center justify-center">
                <card.icon size={24} />
              </div>
              <div>
                <p className="eyebrow mb-1">{card.label}</p>
                <p className="text-xl font-bold text-slate-900 dark:text-white leading-none">{card.value}</p>
              </div>
            </div>
          ))}
        </div>

        <div className="grid gap-8 lg:grid-cols-3 items-start">
          <div className="lg:col-span-2 space-y-8">
            {/* Pending Approval Table */}
            <div className="surface-card p-0 overflow-hidden">
              <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800">
                <h2 className="text-sm font-semibold text-slate-900 dark:text-white">Pending Approval</h2>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse min-w-[500px]">
                  <thead>
                    <tr className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">User Identity</th>
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">Requested</th>
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {error ? (
                      <tr>
                        <td colSpan={3} className="px-6 py-8 text-center">
                          <span className="p-tag p-tag-danger">{error}</span>
                        </td>
                      </tr>
                    ) : pendingUsers.length === 0 ? (
                      <tr>
                        <td colSpan={3} className="px-6 py-12 text-center text-slate-500 text-sm">
                          <ShieldCheck size={32} className="mx-auto text-teal-600 mb-3 opacity-50" />
                          No registrations currently pending review.
                        </td>
                      </tr>
                    ) : (
                      pendingUsers.map((user) => (
                        <tr key={user.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-3">
                              <div className="h-8 w-8 rounded bg-orange-50 text-orange-600 dark:bg-orange-900/20 flex items-center justify-center">
                                <UserRoundCheck size={16} />
                              </div>
                              <div>
                                <p className="text-sm font-semibold text-slate-900 dark:text-white leading-tight">{user.username}</p>
                                <p className="text-xs text-slate-500 mt-0.5">{user.email}</p>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-400">
                            {new Date(user.created_at).toLocaleDateString()}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <form action={approvePendingUser}>
                              <input type="hidden" name="userId" value={user.id} />
                              <button type="submit" className="button-primary h-8 px-4">
                                Approve <ArrowRight size={14} className="ml-1" />
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

            {/* Pending Activation Table */}
            <div className="surface-card p-0 overflow-hidden">
              <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center gap-2">
                <AlertTriangle size={16} className="text-amber-500" />
                <h2 className="text-sm font-semibold text-slate-900 dark:text-white">Pending Activation</h2>
                <span className="text-xs text-slate-500 ml-1">Approved but not yet activated — email may have failed</span>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse min-w-[500px]">
                  <thead>
                    <tr className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">User Identity</th>
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">Approved</th>
                      <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {approvedUsers.length === 0 ? (
                      <tr>
                        <td colSpan={3} className="px-6 py-12 text-center text-slate-500 text-sm">
                          <CheckCircle2 size={32} className="mx-auto text-teal-600 mb-3 opacity-50" />
                          All approved users have activated their accounts.
                        </td>
                      </tr>
                    ) : (
                      approvedUsers.map((user) => (
                        <tr key={user.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors">
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-3">
                              <div className="h-8 w-8 rounded bg-amber-50 text-amber-600 dark:bg-amber-900/20 flex items-center justify-center">
                                <Clock size={16} />
                              </div>
                              <div>
                                <p className="text-sm font-semibold text-slate-900 dark:text-white leading-tight">{user.username}</p>
                                <p className="text-xs text-slate-500 mt-0.5">{user.email}</p>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-400">
                            {new Date(user.created_at).toLocaleDateString()}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <div className="flex items-center justify-end gap-2">
                              <form action={resendActivationEmail}>
                                <input type="hidden" name="userId" value={user.id} />
                                <button type="submit" className="button-ghost h-8 px-3 text-xs" title="Resend activation email">
                                  <RefreshCw size={14} className="mr-1" /> Resend
                                </button>
                              </form>
                              <form action={forceActivateUser}>
                                <input type="hidden" name="userId" value={user.id} />
                                <button type="submit" className="button-primary h-8 px-3 text-xs bg-amber-600 hover:bg-amber-700 border-amber-600" title="Bypass email — activate immediately">
                                  <Zap size={14} className="mr-1" /> Force Activate
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

          <aside className="surface-card">
            <h2 className="section-title mb-4">Security Policy</h2>
            <div className="space-y-4">
              {[
                { title: "Verification", text: "Admin role is strictly required to view or approve requests." },
                { title: "Activation", text: "Approvals immediately dispatch a one-time activation email." },
                { title: "Expiry", text: "Emailed activation tokens expire completely after 24 hours." },
                { title: "Force Activate", text: "Bypasses email verification — use when email delivery fails." },
              ].map((item, i) => (
                <div key={i} className="flex gap-3">
                  <div className="mt-0.5 text-teal-600 shrink-0">
                    <CheckCircle2 size={16} />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-slate-900 dark:text-white leading-none">{item.title}</p>
                    <p className="text-xs text-slate-500 mt-1">{item.text}</p>
                  </div>
                </div>
              ))}
            </div>
          </aside>
        </div>
      </main>
    </div>
  );
}
