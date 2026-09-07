"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

type Workspace = { workspaceId: string; workspaceSlug: string; workspaceName: string; role: string };

export function WorkspaceSwitcher({ currentWorkspaceId }: { currentWorkspaceId: string }) {
  const router = useRouter();
  const [items, setItems] = useState<Workspace[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    fetch("/api/devrelos/api/v1/identity/workspaces", { cache: "no-store" })
      .then((response) => response.ok ? response.json() : [])
      .then((data) => setItems(Array.isArray(data) ? data : []))
      .catch(() => setItems([]));
  }, []);

  if (items.length <= 1) return null;

  async function switchWorkspace(workspaceId: string) {
    if (!workspaceId || workspaceId === currentWorkspaceId) return;
    setBusy(true);
    try {
      const response = await fetch("/api/session/workspace", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ workspaceId })
      });
      if (response.ok) {
        router.replace("/");
        router.refresh();
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <label className="workspace-switcher">
      <span>Workspace</span>
      <select value={currentWorkspaceId} disabled={busy} onChange={(event) => switchWorkspace(event.target.value)}>
        {items.map((item) => <option key={item.workspaceId} value={item.workspaceId}>{item.workspaceName} · {item.role}</option>)}
      </select>
    </label>
  );
}
