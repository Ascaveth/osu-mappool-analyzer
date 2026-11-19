"use client";

import { useState, type FormEvent } from "react";
import { ApiError } from "@/lib/api/client";
import type { SheetImportClient, SheetImportRow } from "@/lib/api/sheet-import";

export function SheetImport({ client, tournamentId, refresh, disabled }: {
  client?: SheetImportClient;
  tournamentId: string;
  refresh: () => Promise<void>;
  disabled: boolean;
}) {
  const [url, setUrl] = useState("");
  const [rows, setRows] = useState<SheetImportRow[] | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const [pending, setPending] = useState<"review" | "save" | null>(null);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");

  const changes = rows?.filter((row) => !row.unchanged && !row.error) ?? [];
  const replacements = changes.filter((row) => row.previous_id).length;
  const hasErrors = rows?.some((row) => row.error);
  const busy = disabled || pending !== null;

  async function review(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!client || busy) return;
    setPending("review");
    setRows(null);
    setConfirmed(false);
    setError("");
    setStatus("");
    try {
      const preview = await client.review(tournamentId, url.trim());
      setRows(preview);
      setStatus(preview.length ? `Review ready. ${preview.length} map rows.` : "This tab has no maps to import.");
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.detail : "We couldn't review the import. Check your connection and try again.");
    } finally {
      setPending(null);
    }
  }

  async function save() {
    if (!client || busy || !rows || hasErrors || !changes.length || (replacements > 0 && !confirmed)) return;
    setPending("save");
    setError("");
    setStatus("");
    try {
      const assignments = rows.map(({ row, slot_id, map_id, previous_id, checksum }) =>
        ({ row, slot_id, map_id, previous_id, checksum }));
      const { saved } = await client.save(tournamentId, assignments);
      setRows(null);
      setConfirmed(false);
      setStatus(saved === 1 ? "Saved 1 assignment." : `Saved ${saved} assignments.`);
      await refresh();
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 409) {
        setRows(null);
        setConfirmed(false);
      }
      setError(cause instanceof ApiError ? cause.detail : "We couldn't confirm the save. Review the import again to check your pool.");
    } finally {
      setPending(null);
    }
  }

  return (
    <details className="pool-bulk-paste sheet-import">
      <summary className="pool-bulk-label" id="sheet-import-title">Import from Google Sheets</summary>
      {client ? (
        <>
          <form aria-labelledby="sheet-import-title" onSubmit={review}>
            <label className="pool-bulk-label" htmlFor="sheet-url">Google Sheets tab link</label>
            <p className="pool-bulk-hint" id="sheet-help">
              Copy the link from the tab you want to import. Anyone with the link must be able to view it.
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <input
                className="input w-full sm:flex-1 min-h-12 text-base"
                id="sheet-url"
                name="url"
                type="url"
                required
                maxLength={2048}
                autoCorrect="off"
                spellCheck={false}
                autoCapitalize="none"
                aria-describedby="sheet-help"
                value={url}
                disabled={busy}
                onChange={(event) => {
                  setUrl(event.target.value);
                  setRows(null);
                  setError("");
                  setStatus("");
                  setConfirmed(false);
                }}
              />
              <button className="btn min-h-12" type="submit" disabled={busy}>
                {pending === "review" ? "Checking sheet and beatmaps…" : "Review import"}
              </button>
            </div>
          </form>
          <p role="status">{status}</p>
          {error && <p className="alert alert-error alert-soft" role="alert">{error}</p>}
          {rows && rows.length > 0 && (
            <>
              <div className="overflow-x-auto">
                <table className="table">
                  <caption className="sr-only">Google Sheets import preview</caption>
                  <thead>
                    <tr>
                      <th scope="col">Row</th>
                      <th scope="col">Target</th>
                      <th scope="col">Beatmap</th>
                      <th scope="col">Result</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.row}>
                        <td>{row.row}</td>
                        <td>{[row.stage, row.slot].filter(Boolean).join(" / ") || "Unmapped"}</td>
                        <td>{[row.title, row.map_id > 0 && `ID ${row.map_id}`].filter(Boolean).join(" · ") || "Missing beatmap"}</td>
                        <td>
                          <p>{row.error || (row.unchanged ? "Already assigned" : row.previous ? `Replaces ${row.previous}` : "Ready to assign")}</p>
                          {row.warning && <p>{row.warning}</p>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {hasErrors ? <p>Fix the flagged rows, then review the import again.</p> : (
                changes.length > 0 && (
                  <>
                    {replacements > 0 && (
                      <label className="flex min-h-12 items-center gap-2">
                        <input type="checkbox" className="checkbox" checked={confirmed} disabled={busy}
                          onChange={(event) => setConfirmed(event.target.checked)} />
                        {replacements === 1 ? "Replace the existing beatmap in 1 slot" : `Replace the existing beatmaps in ${replacements} slots`}
                      </label>
                    )}
                    <button className="btn min-h-12" type="button" onClick={save}
                      disabled={busy || (replacements > 0 && !confirmed)}>
                      {pending === "save" ? "Saving…" : `Save ${changes.length} ${changes.length === 1 ? "assignment" : "assignments"}`}
                    </button>
                  </>
                )
              )}
            </>
          )}
        </>
      ) : <p>Google Sheets import needs the backend and is unavailable in demo mode.</p>}
    </details>
  );
}
