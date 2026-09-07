"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const statuses = ["draft", "needs_work", "ready", "submitted", "accepted", "rejected", "withdrawn"];

export default function SubmissionStatus({ id, initialStatus }: { id: string; initialStatus: string }) {
  const router = useRouter();
  const [status, setStatus] = useState(initialStatus);
  const [saving, setSaving] = useState(false);

  async function update(nextStatus: string) {
    const previous = status;
    setStatus(nextStatus);
    setSaving(true);
    try {
      const response = await fetch(`${apiURL}/api/v1/submissions/${id}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: nextStatus })
      });
      if (!response.ok) throw new Error("Unable to update submission");
      router.refresh();
    } catch {
      setStatus(previous);
    } finally {
      setSaving(false);
    }
  }

  return (
    <select
      aria-label="Submission status"
      className={status === "accepted" ? "status-select accepted" : "status-select"}
      disabled={saving}
      value={status}
      onChange={(event) => update(event.target.value)}
    >
      {statuses.map((value) => (
        <option key={value} value={value}>{value.replaceAll("_", " ")}</option>
      ))}
    </select>
  );
}
