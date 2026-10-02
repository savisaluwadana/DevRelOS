"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

/**
 * Shared delete control for any resource row: confirms, issues the DELETE,
 * surfaces a 409 "still referenced by…" message inline instead of failing
 * silently, and refreshes the page data on success.
 */
export function DeleteButton({
  url,
  confirmMessage,
  label = "Delete",
  onDeleted
}: {
  url: string;
  confirmMessage: string;
  label?: string;
  onDeleted?: () => void;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function remove() {
    if (typeof window !== "undefined" && !window.confirm(confirmMessage)) return;
    setBusy(true);
    setError("");
    try {
      const response = await fetch(url, { method: "DELETE" });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body?.error ?? "Request failed");
      }
      onDeleted?.();
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <span className="delete-action">
      <button className="danger-button small-button" type="button" onClick={remove} disabled={busy}>
        {busy ? "Deleting…" : label}
      </button>
      {error ? <span className="form-error">{error}</span> : null}
    </span>
  );
}
