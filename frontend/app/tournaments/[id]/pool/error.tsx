"use client";

import Link from "next/link";
import { useEffect } from "react";

export default function PoolError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="programme">
      <div className="alert" role="alert">
        <span className="alert-icon" aria-hidden="true">▲</span>
        <p className="alert-text">
          Something went wrong loading this pool. Please try again.
        </p>
      </div>
      <div style={{ display: "flex", gap: "1rem", marginTop: "1rem" }}>
        <button className="btn btn-primary" onClick={reset}>
          Try again
        </button>
        <Link href="/" className="btn btn-ghost">
          Back to tournaments
        </Link>
      </div>
    </main>
  );
}
