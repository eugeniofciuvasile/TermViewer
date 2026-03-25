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
    { label: "Fleet Nodes", value: machines.length, icon: Cpu },
    { label: "Online", value: machines.filter(m => m.status === "online" || m.status === "waiting").length, icon: Wifi },
    { label: "Active Streams", value: machines.filter(m => m.status === "streaming").length, icon: Activity },
  ], [machines]);

  if (status === "loading") return (
    <div className="page-shell">
      <SiteHeader />
      <div className="flex-1 flex flex-col items-center justify-center">
        <LoaderCircle size={32} className="animate-spin text-teal-600 mb-4" />
        <p className="eyebrow">Syncing Controller</p>
      </div>
    </div>
  );

  if (!session) return (
    <div className="page-shell">
      <SiteHeader />
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="surface-card max-w-sm w-full p-8 text-center border-t-4 border-teal-600">
          <Shield size={48} className="mx-auto text-slate-300 dark:text-slate-700 mb-6" />
          <h1 className="section-title">Access Restricted</h1>
          <p className="section-copy mt-2">Authorisation is required to access the terminal fleet controller.</p>
          <SignInButton className="w-full mt-8" />
        </div>
      </div>
    </div>
  );

  return (
    <div className="page-shell">
      <SiteHeader />

      <main className="page-content">
        <div className="mb-6">
          <p className="eyebrow mb-1">Terminal Controller</p>
          <h1 className="page-title">Fleet Overview</h1>
          <p className="section-copy">Manage and monitor your remote relay nodes</p>
        </div>

        {/* Action Bar */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6">
          <div className="relative w-full md:w-96">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={16} />
            <input
              type="text"
              placeholder="Search fleet by name or ID..."
              className="input-field pl-10"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
          
          <div className="flex items-center gap-3">
            <button onClick={() => { setNewMachineCreds(null); setIsCreateModalOpen(true); }} className="button-primary">
              <Plus size={16} />
              Register Machine
            </button>
            <button onClick={() => void fetchMachines()} className="button-secondary p-0 w-9" title="Refresh">
              <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
            </button>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 mb-8">
          {stats.map((stat) => (
            <div key={stat.label} className="surface-card flex items-center gap-4">
              <div className="h-12 w-12 rounded-lg bg-teal-50 text-teal-600 dark:bg-teal-900/20 dark:text-teal-400 flex items-center justify-center">
                <stat.icon size={24} />
              </div>
              <div>
                <p className="eyebrow mb-1">{stat.label}</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white leading-none">{stat.value}</p>
              </div>
            </div>
          ))}
        </div>

        {shareError && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/10 text-red-600 dark:text-red-400 text-sm rounded-md border border-red-100 dark:border-red-900/20 flex items-center gap-3">
            <Shield size={18} />
            <span className="font-semibold">{shareError}</span>
          </div>
        )}

        {/* DataTable */}
        <div className="surface-card p-0 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse min-w-[700px]">
              <thead>
                <tr className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                  <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">Identified Node</th>
                  <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">Global Status</th>
                  <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider">Presence</th>
                  <th className="px-6 py-4 text-xs font-semibold text-slate-500 uppercase tracking-wider text-right">Operations</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {filteredMachines.length === 0 && !loading ? (
                  <tr>
                    <td colSpan={4} className="px-6 py-12 text-center text-slate-500 text-sm">
                      No nodes found matching your criteria.
                    </td>
                  </tr>
                ) : (
                  filteredMachines.map((machine) => {
                    const isOnline = machine.status === "online" || machine.status === "waiting";
                    const isStreaming = machine.status === "streaming";
                    const statusSeverity = isOnline ? "p-tag-success" : isStreaming ? "p-tag-warning" : "p-tag-danger";

                    return (
                      <tr key={machine.id} className="hover:bg-slate-50/50 dark:hover:bg-slate-800/30 transition-colors group">
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <div className="h-8 w-8 rounded bg-slate-100 dark:bg-slate-800 flex items-center justify-center text-slate-500">
                              <Cpu size={16} />
                            </div>
                            <div>
                              <p className="text-sm font-semibold text-slate-900 dark:text-white leading-tight">{machine.name}</p>
                              <p className="text-[11px] font-mono text-slate-500 mt-0.5">{machine.client_id}</p>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <span className={`p-tag ${statusSeverity}`}>
                            {machine.status}
                          </span>
                        </td>
                        <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-400">
                          <div className="flex items-center gap-2">
                            <Clock size={14} className="opacity-70" />
                            {formatTimestamp(machine.last_seen_at)}
                          </div>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-2 relative">
                            <button
                              disabled={machine.status === "offline" || shareLoadingMachineId === machine.id}
                              onClick={() => void createShareSession(machine)}
                              className="button-ghost h-8 px-3"
                            >
                              {shareLoadingMachineId === machine.id ? <RefreshCw size={14} className="animate-spin" /> : <QrCode size={14} />}
                              <span className="ml-2 hidden sm:inline">Share</span>
                            </button>
                            
                            <button 
                              onClick={() => setMachineForActions(machine)}
                              className="button-ghost p-0 w-8 h-8 rounded-full"
                              title="More actions"
                            >
                              <MoreVertical size={14} />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>

        <div className="mt-6 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-4">
          <div className="flex items-center gap-2">
            <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
            {lastRefreshedAt ? `Last Sync: ${lastRefreshedAt.toLocaleTimeString()}` : "Ready"}
          </div>
          <div className="flex items-center gap-4">
            <a href="#" className="hover:text-slate-900 dark:hover:text-white flex items-center gap-1"><ExternalLink size={12} /> Docs</a>
            <span>TermViewer Control Plane v0.1.0</span>
          </div>
        </div>
      </main>

      {/* Action Picker Modal */}
      {machineForActions && (
        <div className="dialog-backdrop" onClick={() => setMachineForActions(null)}>
          <div className="dialog-card max-w-[280px] p-0 overflow-hidden" onClick={e => e.stopPropagation()}>
            <div className="p-4 bg-slate-50 dark:bg-slate-800/50 border-b border-slate-100 dark:border-slate-800">
              <p className="text-[10px] font-bold text-slate-500 uppercase tracking-widest mb-1">Actions</p>
              <h3 className="text-sm font-bold text-slate-900 dark:text-white truncate">{machineForActions.name}</h3>
            </div>
            <div className="p-2 space-y-1">
              <button 
                onClick={() => { setMachineToRename(machineForActions); setNewName(machineForActions.name); setMachineForActions(null); }}
                className="w-full flex items-center gap-3 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-md transition-colors"
              >
                <div className="w-8 h-8 rounded bg-teal-50 dark:bg-teal-900/20 text-teal-600 flex items-center justify-center"><Pencil size={16} /></div>
                Rename Node
              </button>
              <button 
                onClick={() => { setMachineToRegenerate(machineForActions); setRegeneratedSecret(null); setMachineForActions(null); }}
                className="w-full flex items-center gap-3 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 rounded-md transition-colors"
              >
                <div className="w-8 h-8 rounded bg-orange-50 dark:bg-orange-900/20 text-orange-600 flex items-center justify-center"><KeyRound size={16} /></div>
                Reset Secret
              </button>
              <div className="h-px bg-slate-100 dark:bg-slate-800 my-1 mx-2" />
              <button 
                onClick={() => { setMachineToDelete(machineForActions); setMachineForActions(null); }}
                className="w-full flex items-center gap-3 px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md transition-colors"
              >
                <div className="w-8 h-8 rounded bg-red-50 dark:bg-red-900/20 text-red-600 flex items-center justify-center"><Trash2 size={16} /></div>
                Delete Node
              </button>
            </div>
            <button onClick={() => setMachineForActions(null)} className="w-full py-3 text-xs font-bold text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 bg-slate-50/50 dark:bg-slate-800/20 transition-colors">
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Rename Modal */}
      {machineToRename && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Rename Machine</h2>
              <button onClick={() => setMachineToRename(null)} className="text-slate-400 hover:text-slate-600"><X size={20} /></button>
            </div>
            <form onSubmit={handleRenameMachine} className="space-y-6">
              <div>
                <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">New Identity</label>
                <input type="text" required autoFocus value={newName} onChange={(e) => setNewName(e.target.value)} className="input-field" placeholder="Machine Name" />
              </div>
              <div className="flex justify-end gap-3">
                <button type="button" onClick={() => setMachineToRename(null)} className="button-ghost">Cancel</button>
                <button type="submit" disabled={isActionLoading} className="button-primary">
                  {isActionLoading ? <LoaderCircle size={16} className="animate-spin" /> : "Save Changes"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Regenerate Secret Modal */}
      {machineToRegenerate && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Security Key Reset</h2>
              <button onClick={() => setMachineToRegenerate(null)} className="text-slate-400 hover:text-slate-600"><X size={20} /></button>
            </div>
            {!regeneratedSecret ? (
              <div className="space-y-6">
                <div className="p-4 bg-orange-50 dark:bg-orange-950/20 border border-orange-100 dark:border-orange-900/30 rounded-md flex items-start gap-3">
                  <AlertTriangle className="text-orange-600 shrink-0" size={20} />
                  <p className="text-xs text-orange-800 dark:text-orange-300 leading-relaxed">
                    Resetting the secret will immediately disconnect the current agent. You must update the agent configuration with the new key.
                  </p>
                </div>
                <div className="flex justify-end gap-3">
                  <button type="button" onClick={() => setMachineToRegenerate(null)} className="button-ghost">Abort</button>
                  <button onClick={handleRegenerateSecret} disabled={isActionLoading} className="button-primary bg-orange-600 border-orange-600 hover:bg-orange-700 shadow-orange-600/20">
                    {isActionLoading ? <LoaderCircle size={16} className="animate-spin" /> : "Confirm Reset"}
                  </button>
                </div>
              </div>
            ) : (
              <div className="animate-fade-in space-y-6">
                <div className="p-4 bg-teal-50 dark:bg-teal-900/10 rounded-lg border border-teal-100 dark:border-teal-900/30 flex items-start gap-3">
                  <Shield size={24} className="text-teal-600 shrink-0" />
                  <p className="text-xs text-teal-600 dark:text-teal-500 font-bold">Secret successfully regenerated. Copy it now; it won&apos;t be shown again.</p>
                </div>
                <div className="relative group">
                  <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1.5">New Client Secret</label>
                  <div className="relative">
                    <input type="text" readOnly value={regeneratedSecret} className="input-field pr-10 font-mono text-sm font-bold text-teal-700 bg-teal-50/50 dark:bg-teal-900/20" />
                    <button onClick={() => copyToClipboard("reg", regeneratedSecret)} className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-teal-600 p-1.5">
                      {copiedKey === "reg" ? <Check size={16} /> : <Copy size={16} />}
                    </button>
                  </div>
                </div>
                <button onClick={() => setMachineToRegenerate(null)} className="button-primary w-full">I have saved the key</button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Delete Modal */}
      {machineToDelete && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm border-t-4 border-red-600">
            <h2 className="section-title text-red-600 mb-4">Confirm Deletion</h2>
            <p className="section-copy mb-8">
              Are you absolutely sure you want to remove <span className="font-bold text-slate-900 dark:text-white">{machineToDelete.name}</span>? This action is permanent and cannot be reversed.
            </p>
            <div className="flex justify-end gap-3">
              <button type="button" onClick={() => setMachineToDelete(null)} className="button-ghost">Cancel</button>
              <button onClick={handleDeleteMachine} disabled={isActionLoading} className="button-primary bg-red-600 border-red-600 hover:bg-red-700 shadow-red-600/20">
                {isActionLoading ? <LoaderCircle size={16} className="animate-spin" /> : "Delete Permanently"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Register Modal */}
      {isCreateModalOpen && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-md">
            <div className="flex items-center justify-between mb-6">
              <h2 className="section-title">Register Node</h2>
              <button onClick={() => { setIsCreateModalOpen(false); setNewMachineCreds(null); }} className="text-slate-400 hover:text-slate-600 transition-colors">
                <X size={20} />
              </button>
            </div>

            {!newMachineCreds ? (
              <form onSubmit={handleCreateMachine} className="space-y-6">
                <p className="section-copy">Register a new host agent to this relay realm. Choose a descriptive name to identify this machine in the fleet.</p>
                <div>
                  <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">Node Name</label>
                  <input type="text" required autoFocus value={newMachineName} onChange={(e) => setNewMachineName(e.target.value)} className="input-field" placeholder="e.g. ALPHA-SERVER" />
                </div>
                <div className="flex justify-end gap-3">
                  <button type="button" onClick={() => setIsCreateModalOpen(false)} className="button-ghost">Cancel</button>
                  <button type="submit" className="button-primary">Provision Agent</button>
                </div>
              </form>
            ) : (
              <div className="animate-fade-in space-y-6">
                <div className="p-4 bg-teal-50 dark:bg-teal-900/10 rounded-lg border border-teal-100 dark:border-teal-900/30 flex items-start gap-3">
                  <Shield size={24} className="text-teal-600 shrink-0" />
                  <div>
                    <h3 className="text-sm font-bold text-teal-800 dark:text-teal-300">Credentials Provisioned</h3>
                    <p className="text-xs text-teal-600 dark:text-teal-500 mt-1">Copy these credentials immediately. The secret is securely hashed and cannot be retrieved again.</p>
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1.5">Client ID</label>
                    <div className="relative">
                      <input type="text" readOnly value={newMachineCreds.client_id} className="input-field pr-10 font-mono text-sm bg-slate-50 dark:bg-slate-900" />
                      <button onClick={() => copyToClipboard("cid", newMachineCreds.client_id)} className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-teal-600 p-1.5">
                        {copiedKey === "cid" ? <Check size={16} /> : <Copy size={16} />}
                      </button>
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1.5">Client Secret</label>
                    <div className="relative">
                      <input type="text" readOnly value={newMachineCreds.client_secret} className="input-field pr-10 font-mono text-sm font-bold text-teal-700 bg-teal-50/50 dark:bg-teal-900/20" />
                      <button onClick={() => copyToClipboard("sec", newMachineCreds.client_secret)} className="absolute right-2 top-1/2 -translate-y-1/2 text-teal-600 hover:text-teal-800 p-1.5">
                        {copiedKey === "sec" ? <Check size={16} /> : <Copy size={16} />}
                      </button>
                    </div>
                  </div>
                </div>

                <button onClick={() => { setIsCreateModalOpen(false); setNewMachineCreds(null); }} className="button-primary w-full mt-2">Securely Dismiss</button>
              </div>
            )}
          </div>
        </div>
      )}

      {shareModal && (
        <div className="dialog-backdrop">
          <div className="dialog-card max-w-sm text-center">
            <div className="flex justify-end mb-2">
              <button onClick={() => setShareModal(null)} className="text-slate-400 hover:text-slate-600">
                <X size={20} />
              </button>
            </div>

            <div className="mx-auto mb-6 p-4 bg-white border border-slate-200 rounded-xl shadow-sm w-fit inline-block">
              <QRCodeSVG value={shareModal.deepLink} size={220} includeMargin={true} />
            </div>

            <h2 className="text-lg font-bold text-slate-900 dark:text-white truncate px-4">{shareModal.machineName}</h2>
            <p className="text-sm text-slate-500 mt-1 flex items-center justify-center gap-2">
              <Shield size={14} /> Ephemeral Relay Token
            </p>
            
            <div className="mt-6 text-left">
              <label className="block text-xs font-semibold text-slate-600 dark:text-slate-400 mb-1.5">Deep Link URL</label>
              <div className="relative">
                <input type="text" readOnly value={shareModal.deepLink} className="input-field pr-10 font-mono text-xs bg-slate-50 dark:bg-slate-900" />
                <button onClick={() => copyToClipboard("link", shareModal.deepLink)} className="absolute right-2 top-1/2 -translate-y-1/2 text-teal-600 hover:text-teal-800 p-1.5">
                  {copiedKey === "link" ? <Check size={14} /> : <Copy size={14} />}
                </button>
              </div>
            </div>

            <div className="mt-5 flex items-center justify-center gap-2 text-xs font-semibold text-orange-600 dark:text-orange-400">
              <Clock size={14} /> Expires at {new Date(shareModal.expiresAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
            </div>

            <button onClick={() => setShareModal(null)} className="button-secondary w-full mt-6">Close</button>
          </div>
        </div>
      )}
    </div>
  );
}
