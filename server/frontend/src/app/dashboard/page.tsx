"use client";

import api from "@/lib/api";
import { QRCodeSVG } from "qrcode.react";
import { useSession } from "next-auth/react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Activity,
  Check,
  Clock,
  Copy,
  Cpu,
  LoaderCircle,
  Plus,
  QrCode,
  RefreshCw,
  Shield,
  Wifi,
  X,
  Search,
  MoreVertical,
  Pencil,
  Trash2,
  KeyRound,
  AlertTriangle,
  ExternalLink,
} from "lucide-react";

import { SignInButton } from "@/components/auth-buttons";
import SiteHeader from "@/components/site-header";
import SiteFooter from "@/components/site-footer";

interface ActiveShareSession {
  id: number;
  status: string;
  expires_at: string;
}

interface Machine {
  id: number;
  name: string;
  client_id: string;
  status: string;
  last_seen_at?: string | null;
  active_share_session?: ActiveShareSession | null;
}

interface MachineCredentials {
  client_id: string;
  client_secret: string;
}

interface ShareSessionModalState {
  machineId: number;
  machineName: string;
  deepLink: string;
  expiresAt: string;
}

interface CreateShareSessionResponse {
  session_id: number;
  session_token: string;
  expires_at: string;
  server_url: string;
  deep_link: string;
  status: string;
}

function formatTimestamp(value?: string | null): string {
  if (!value) return "Never";
  const date = new Date(value);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 60000) return "Just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  
  return date.toLocaleDateString([], { month: "short", day: "numeric" }) + " " + date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export default function Dashboard() {
  const { data: session, status } = useSession();
  const [machines, setMachines] = useState<Machine[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newMachineName, setNewMachineName] = useState("");
  const [newMachineCreds, setNewMachineCreds] = useState<MachineCredentials | null>(null);
  const [shareModal, setShareModal] = useState<ShareSessionModalState | null>(null);
  const [shareLoadingMachineId, setShareLoadingMachineId] = useState<number | null>(null);
  const [shareError, setShareError] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Management States
  const [machineForActions, setMachineForActions] = useState<Machine | null>(null);
  const [machineToRename, setMachineToRename] = useState<Machine | null>(null);
  const [newName, setNewName] = useState("");
  const [machineToRegenerate, setMachineToRegenerate] = useState<Machine | null>(null);
  const [regeneratedSecret, setRegeneratedSecret] = useState<string | null>(null);
  const [machineToDelete, setMachineToDelete] = useState<Machine | null>(null);
  const [isActionLoading, setIsActionLoading] = useState(false);

  const fetchMachines = useCallback(async () => {
    if (!session?.accessToken) return;
    try {
      const response = await api.get<Machine[]>("/api/machines");
      setMachines(response.data);
      setLastRefreshedAt(new Date());
    } catch (error) {
      // Error is already sanitized by the API client interceptor
      console.error("Sync failed:", error instanceof Error ? error.message : "Unknown");
    } finally {
      setLoading(false);
    }
  }, [session?.accessToken]);

  useEffect(() => {
    if (session?.accessToken) void fetchMachines();
  }, [fetchMachines, session?.accessToken]);

  useEffect(() => {
    if (!session?.accessToken) return;
    const interval = setInterval(() => {
      if (document.visibilityState === "visible") void fetchMachines();
    }, 15000);
    return () => clearInterval(interval);
  }, [fetchMachines, session?.accessToken]);

  const createShareSession = useCallback(async (machine: Machine) => {
    if (!session?.accessToken) return;
    setShareLoadingMachineId(machine.id);
    setShareError(null);
    try {
      const response = await api.post<CreateShareSessionResponse>(
        `/api/machines/${machine.id}/share-session`
      );
      setShareModal({
        machineId: machine.id,
        machineName: machine.name,
        deepLink: response.data.deep_link,
        expiresAt: response.data.expires_at,
      });
      await fetchMachines();
    } catch (error) {
      setShareError(error instanceof Error ? error.message : "Failed to initiate share session.");
    } finally {
      setShareLoadingMachineId(null);
    }
  }, [fetchMachines, session?.accessToken]);

  const handleCreateMachine = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session?.accessToken) return;
    try {
      const response = await api.post<MachineCredentials>(
        "/api/machines",
        { name: newMachineName }
      );
      setNewMachineCreds(response.data);
      setNewMachineName("");
      await fetchMachines();
    } catch (error) {
      console.error("Provisioning failed:", error);
    }
  };

  const handleRenameMachine = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session?.accessToken || !machineToRename) return;
    setIsActionLoading(true);
    try {
      await api.patch(`/api/machines/${machineToRename.id}`, { name: newName });
      setMachineToRename(null);
      await fetchMachines();
    } catch (error) {
      console.error("Rename failed:", error);
    } finally {
      setIsActionLoading(false);
    }
  };

  const handleRegenerateSecret = async () => {
    if (!session?.accessToken || !machineToRegenerate) return;
    setIsActionLoading(true);
    try {
      const response = await api.post(`/api/machines/${machineToRegenerate.id}/regenerate-secret`);
      setRegeneratedSecret(response.data.client_secret);
    } catch (error) {
      console.error("Regenerate failed:", error);
    } finally {
      setIsActionLoading(false);
    }
  };

  const handleDeleteMachine = async () => {
    if (!session?.accessToken || !machineToDelete) return;
    setIsActionLoading(true);
    try {
      await api.delete(`/api/machines/${machineToDelete.id}`);
      setMachineToDelete(null);
      await fetchMachines();
    } catch (error) {
      console.error("Delete failed:", error);
    } finally {
      setIsActionLoading(false);
    }
  };

  const copyToClipboard = async (key: string, text: string) => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
      } else {
        const textArea = document.createElement("textarea");
        textArea.value = text;
        textArea.style.position = "fixed";
        textArea.style.left = "-999999px";
        textArea.style.top = "-999999px";
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        document.execCommand("copy");
        textArea.remove();
      }
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(null), 2000);
    } catch (err) {
      console.error("Clipboard failed", err);
    }
  };

  const filteredMachines = useMemo(() => {
    return machines.filter(m => 
      m.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
      m.client_id.toLowerCase().includes(searchQuery.toLowerCase())
    );
  }, [machines, searchQuery]);

  const stats = useMemo(() => [
    { label: "Fleet nodes", value: machines.length, icon: Cpu },
    { label: "Online", value: machines.filter(m => m.status === "online" || m.status === "waiting").length, icon: Wifi },
    { label: "Streaming", value: machines.filter(m => m.status === "streaming").length, icon: Activity },
  ], [machines]);

  if (status === "loading") return (
    <div className="page-shell">
      <SiteHeader />
      <div className="flex-1 flex flex-col items-center justify-center">
        <LoaderCircle size={24} className="animate-spin text-[var(--accent)] mb-3" />
        <p className="eyebrow">syncing</p>
      </div>
    </div>
  );

  if (!session) return (
    <div className="page-shell">
      <SiteHeader />
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="max-w-sm w-full text-center">
          <Shield size={32} className="mx-auto text-[var(--text-muted)] mb-4" />
          <h1 className="section-title">Access restricted</h1>
          <p className="section-copy mt-2">Authorisation is required to access the terminal fleet controller.</p>
          <SignInButton className="w-full mt-6" />
        </div>
      </div>
    </div>
  );


  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content">
        {/* ── Header ── */}
        <div className="flex flex-col gap-6 mb-8">
          <div>
            <p className="eyebrow mb-2">fleet</p>
            <h1 className="page-title">Machines</h1>
            <p className="section-copy mt-1">Manage and monitor your remote relay nodes.</p>
          </div>

          {/* Stats row */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {stats.map((stat, i) => (
              <div key={stat.label} className="stat-card" style={{ animationDelay: `${i * 60}ms` }}>
                <div className="flex items-center gap-3 mb-3">
                  <div className="h-8 w-8 rounded-lg bg-[var(--accent-muted)] flex items-center justify-center">
                    <stat.icon size={16} className="text-[var(--accent)]" />
                  </div>
                  <span className="eyebrow">{stat.label}</span>
                </div>
                <p className="text-2xl font-semibold text-[var(--text-primary)] leading-none tracking-tight">{stat.value}</p>
              </div>
            ))}
          </div>
        </div>

        {/* ── Floating toolbar ── */}
        <div className="toolbar mb-6">
          <div className="relative w-full sm:w-auto sm:flex-1 min-w-0">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] pointer-events-none" size={16} />
            <input
              type="text"
              placeholder="Search machines…"
              className="input-field pl-11"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          <div className="hidden sm:block h-5 w-px bg-[var(--border)]" />
          <div className="flex items-center gap-2 ml-auto">
            <button
              onClick={() => void fetchMachines()}
              className="button-icon"
              title="Refresh"
            >
              <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
            </button>
            <button
              onClick={() => { setNewMachineCreds(null); setIsCreateModalOpen(true); }}
              className="button-primary button-sm"
            >
              <Plus size={15} /> Register
            </button>
          </div>
        </div>

        {shareError && (
          <div className="alert alert-danger mb-5">
            <Shield size={16} className="shrink-0 mt-0.5" />
            <span className="font-medium">{shareError}</span>
          </div>
        )}

        {/* ── Machine card grid ── */}
        <div className="grid gap-3">
          {filteredMachines.length === 0 && !loading ? (
            <div className="surface-card text-center py-16">
              <Cpu size={32} className="mx-auto text-[var(--text-muted)] mb-3 opacity-40" />
              <p className="text-sm text-[var(--text-muted)]">No machines found.</p>
              <button
                onClick={() => { setNewMachineCreds(null); setIsCreateModalOpen(true); }}
                className="button-primary mt-4"
              >
                <Plus size={15} /> Register your first machine
              </button>
            </div>
          ) : (
            filteredMachines.map((machine, i) => {
              const isOnline = machine.status === "online" || machine.status === "waiting";
              const isStreaming = machine.status === "streaming";
              const dotClass = isOnline ? "status-dot-online" : isStreaming ? "status-dot-streaming" : "status-dot-offline";
              const tagClass = isOnline ? "p-tag-success" : isStreaming ? "p-tag-warning" : "p-tag-danger";

              return (
                <div
                  key={machine.id}
                  className="machine-card animate-slide-up"
                  style={{ animationDelay: `${i * 40}ms`, animationFillMode: "backwards" }}
                >
                  <div className="flex items-center gap-4">
                    {/* Status dot + identity */}
                    <div className="flex items-center gap-3.5 flex-1 min-w-0">
                      <div className={`status-dot ${dotClass}`} />
                      <div className="min-w-0">
                        <p className="text-[15px] font-semibold text-[var(--text-primary)] truncate leading-tight">
                          {machine.name}
                        </p>
                        <p className="text-xs font-mono text-[var(--text-muted)] mt-1 truncate">
                          {machine.client_id}
                        </p>
                      </div>
                    </div>

                    {/* Status + last seen (hidden on mobile) */}
                    <div className="hidden sm:flex items-center gap-5 shrink-0">
                      <span className={`p-tag ${tagClass}`}>
                        {machine.status}
                      </span>
                      <span className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] font-mono w-24 justify-end">
                        <Clock size={12} />
                        {formatTimestamp(machine.last_seen_at)}
                      </span>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-1.5 shrink-0">
                      <button
                        disabled={machine.status === "offline" || shareLoadingMachineId === machine.id}
                        onClick={() => void createShareSession(machine)}
                        className="button-ghost h-9 px-3"
                        title="Share via QR"
                      >
                        {shareLoadingMachineId === machine.id
                          ? <RefreshCw size={15} className="animate-spin" />
                          : <QrCode size={15} />}
                        <span className="hidden md:inline ml-1">Share</span>
                      </button>
                      <button
                        onClick={() => setMachineForActions(machine)}
                        className="button-ghost h-9 w-9 p-0"
                        title="More actions"
                      >
                        <MoreVertical size={15} />
                      </button>
                    </div>
                  </div>

                  {/* Mobile-only status row */}
                  <div className="flex items-center gap-3 mt-3 sm:hidden">
                    <span className={`p-tag ${tagClass}`}>{machine.status}</span>
                    <span className="flex items-center gap-1 text-[11px] text-[var(--text-muted)] font-mono">
                      <Clock size={11} /> {formatTimestamp(machine.last_seen_at)}
                    </span>
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* ── Footer bar ── */}
        <div className="mt-6 flex items-center justify-between text-[11px] text-[var(--text-muted)] font-mono">
          <span className="flex items-center gap-1.5">
            <RefreshCw size={10} className={loading ? "animate-spin" : ""} />
            {lastRefreshedAt ? `sync ${lastRefreshedAt.toLocaleTimeString()}` : "ready"}
          </span>
          <span className="flex items-center gap-3">
            <a href="https://github.com/eugeniofciuvasile/TermViewer/tree/main/docs" target="_blank" rel="noopener noreferrer" className="hover:text-[var(--text-primary)] transition-colors flex items-center gap-1">
              <ExternalLink size={10} /> docs
            </a>
            <span>v0.1.0</span>
          </span>
        </div>
      </main>

      <SiteFooter />

      {/* ══════════════════════════════════════════════
          MODALS — floating glass panels with scale-in
          ══════════════════════════════════════════════ */}

      {/* Action context menu */}
      {machineForActions && (
        <div className="dialog-backdrop" onClick={() => setMachineForActions(null)}>
          <div className="context-menu w-[240px]" onClick={e => e.stopPropagation()}>
            <div className="px-4 py-3">
              <p className="eyebrow mb-0.5">actions</p>
              <p className="text-sm font-medium text-[var(--text-primary)] truncate">{machineForActions.name}</p>
            </div>
            <div className="context-menu-divider" />
            <div className="py-1">
              <button
                onClick={() => { setMachineToRename(machineForActions); setNewName(machineForActions.name); setMachineForActions(null); }}
                className="context-menu-item"
              >
                <Pencil size={15} /> Rename
              </button>
              <button
                onClick={() => { setMachineToRegenerate(machineForActions); setRegeneratedSecret(null); setMachineForActions(null); }}
                className="context-menu-item"
              >
                <KeyRound size={15} /> Reset secret
              </button>
              <div className="context-menu-divider" />
              <button
                onClick={() => { setMachineToDelete(machineForActions); setMachineForActions(null); }}
                className="context-menu-item context-menu-item-danger"
              >
                <Trash2 size={15} /> Delete machine
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Rename modal */}
      {machineToRename && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Rename machine</h2>
              <button onClick={() => setMachineToRename(null)} className="button-ghost h-8 w-8 p-0"><X size={16} /></button>
            </div>
            <form onSubmit={handleRenameMachine} className="space-y-5">
              <div>
                <label className="eyebrow block mb-2">new name</label>
                <input type="text" required autoFocus value={newName} onChange={(e) => setNewName(e.target.value)} className="input-field" placeholder="Machine name" />
              </div>
              <div className="flex justify-end gap-2.5 pt-2">
                <button type="button" onClick={() => setMachineToRename(null)} className="button-ghost">Cancel</button>
                <button type="submit" disabled={isActionLoading} className="button-primary">
                  {isActionLoading ? <LoaderCircle size={15} className="animate-spin" /> : "Save"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Regenerate secret modal */}
      {machineToRegenerate && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Reset secret</h2>
              <button onClick={() => setMachineToRegenerate(null)} className="button-ghost h-8 w-8 p-0"><X size={16} /></button>
            </div>
            {!regeneratedSecret ? (
              <div className="space-y-5">
                <div className="alert alert-warning">
                  <AlertTriangle size={16} className="shrink-0 mt-0.5" />
                  <p className="text-sm">This will disconnect the current agent. Update your configuration with the new key.</p>
                </div>
                <div className="flex justify-end gap-2.5 pt-2">
                  <button type="button" onClick={() => setMachineToRegenerate(null)} className="button-ghost">Abort</button>
                  <button onClick={handleRegenerateSecret} disabled={isActionLoading} className="button-primary" style={{ background: 'var(--warning)', borderColor: 'var(--warning)' }}>
                    {isActionLoading ? <LoaderCircle size={15} className="animate-spin" /> : "Confirm reset"}
                  </button>
                </div>
              </div>
            ) : (
              <div className="animate-scale-in space-y-5">
                <div className="alert alert-success">
                  <Shield size={16} className="shrink-0 mt-0.5" />
                  <p className="text-sm font-medium">Secret regenerated. Copy it now — it won&apos;t be shown again.</p>
                </div>
                <div>
                  <label className="eyebrow block mb-2">new secret</label>
                  <div className="relative">
                    <input type="text" readOnly value={regeneratedSecret} className="input-field pr-11 font-mono" />
                    <button onClick={() => copyToClipboard("reg", regeneratedSecret)} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors p-1.5">
                      {copiedKey === "reg" ? <Check size={15} /> : <Copy size={15} />}
                    </button>
                  </div>
                </div>
                <button onClick={() => setMachineToRegenerate(null)} className="button-primary w-full">Done</button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Delete confirmation modal */}
      {machineToDelete && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm">
            <div className="flex items-center gap-3 mb-5">
              <div className="flex items-center justify-center h-10 w-10 rounded-full bg-[var(--danger-muted)]">
                <Trash2 size={18} className="text-[var(--danger)]" />
              </div>
              <div>
                <h2 className="section-title">Delete machine</h2>
                <p className="text-xs text-[var(--text-muted)] mt-0.5">This action is permanent.</p>
              </div>
            </div>
            <p className="section-copy mb-6">
              Remove <span className="font-semibold text-[var(--text-primary)]">{machineToDelete.name}</span> and all associated sessions?
            </p>
            <div className="flex justify-end gap-2.5">
              <button type="button" onClick={() => setMachineToDelete(null)} className="button-ghost">Cancel</button>
              <button onClick={handleDeleteMachine} disabled={isActionLoading} className="button-primary" style={{ background: 'var(--danger)', borderColor: 'var(--danger)' }}>
                {isActionLoading ? <LoaderCircle size={15} className="animate-spin" /> : "Delete permanently"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Register machine modal */}
      {isCreateModalOpen && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Register machine</h2>
              <button onClick={() => { setIsCreateModalOpen(false); setNewMachineCreds(null); }} className="button-ghost h-8 w-8 p-0">
                <X size={16} />
              </button>
            </div>

            {!newMachineCreds ? (
              <form onSubmit={handleCreateMachine} className="space-y-5">
                <p className="section-copy">Register a new host agent. Choose a descriptive name to identify this machine.</p>
                <div>
                  <label className="eyebrow block mb-2">node name</label>
                  <input type="text" required autoFocus value={newMachineName} onChange={(e) => setNewMachineName(e.target.value)} className="input-field" placeholder="e.g. alpha-server" />
                </div>
                <div className="flex justify-end gap-2.5 pt-2">
                  <button type="button" onClick={() => setIsCreateModalOpen(false)} className="button-ghost">Cancel</button>
                  <button type="submit" className="button-primary">Provision</button>
                </div>
              </form>
            ) : (
              <div className="animate-scale-in space-y-5">
                <div className="alert alert-success">
                  <Shield size={16} className="shrink-0 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Credentials provisioned</p>
                    <p className="text-xs opacity-80 mt-0.5">Copy now. The secret is hashed and cannot be retrieved.</p>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="eyebrow block mb-2">client id</label>
                    <div className="relative">
                      <input type="text" readOnly value={newMachineCreds.client_id} className="input-field pr-11 font-mono" />
                      <button onClick={() => copyToClipboard("cid", newMachineCreds.client_id)} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors p-1.5">
                        {copiedKey === "cid" ? <Check size={15} /> : <Copy size={15} />}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="eyebrow block mb-2">client secret</label>
                    <div className="relative">
                      <input type="text" readOnly value={newMachineCreds.client_secret} className="input-field pr-11 font-mono" />
                      <button onClick={() => copyToClipboard("sec", newMachineCreds.client_secret)} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors p-1.5">
                        {copiedKey === "sec" ? <Check size={15} /> : <Copy size={15} />}
                      </button>
                    </div>
                  </div>
                </div>

                <button onClick={() => { setIsCreateModalOpen(false); setNewMachineCreds(null); }} className="button-primary w-full">Done</button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* QR Share modal */}
      {shareModal && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm text-center">
            <div className="flex justify-end mb-1">
              <button onClick={() => setShareModal(null)} className="button-ghost h-8 w-8 p-0"><X size={16} /></button>
            </div>

            <div className="mx-auto mb-6 p-5 bg-white rounded-2xl shadow-[var(--shadow-md)] w-fit">
              <QRCodeSVG value={shareModal.deepLink} size={200} includeMargin={true} />
            </div>

            <h2 className="text-lg font-semibold text-[var(--text-primary)] truncate px-4">{shareModal.machineName}</h2>
            <p className="text-xs text-[var(--text-muted)] mt-1 flex items-center justify-center gap-1.5 font-mono">
              <Shield size={12} /> ephemeral relay token
            </p>

            <div className="mt-6 text-left">
              <label className="eyebrow block mb-2">deep link</label>
              <div className="relative">
                <input type="text" readOnly value={shareModal.deepLink} className="input-field pr-11 font-mono text-xs" />
                <button onClick={() => copyToClipboard("link", shareModal.deepLink)} className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--accent)] transition-colors p-1.5">
                  {copiedKey === "link" ? <Check size={13} /> : <Copy size={13} />}
                </button>
              </div>
            </div>

            <div className="mt-5 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[var(--warning-muted)] text-xs font-mono text-[var(--warning)]">
              <Clock size={12} /> expires {new Date(shareModal.expiresAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
            </div>

            <button onClick={() => setShareModal(null)} className="button-secondary w-full mt-6">Close</button>
          </div>
        </div>
      )}
    </div>
  );
}
