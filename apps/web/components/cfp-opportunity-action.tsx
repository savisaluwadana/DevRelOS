"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function CreateSubmissionDraftButton({ cfpId, talkId, disabled = false }: { cfpId: string; talkId: string; disabled?: boolean }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function createDraft() {
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch(`${apiURL}/api/v1/submissions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ cfpId, talkId, status: "draft", notes: "Created from DevRelOS opportunity ranking." })
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Could not create submission draft" }));
        throw new Error(payload.error ?? "Could not create submission draft");
      }
      setMessage("Draft created.");
      router.refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Could not create submission draft");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="opportunity-action-stack">
      <button className="button primary" type="button" disabled={disabled || busy} onClick={createDraft}>
        {disabled ? "Submission exists" : busy ? "Creating…" : "Create submission draft"}
      </button>
      {message && <span className="action-note">{message}</span>}
    </div>
  );
}
