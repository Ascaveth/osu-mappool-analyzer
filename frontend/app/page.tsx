import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { HowToUse } from "@/components/HowToUse";
import { WipDisclaimer } from "@/components/WipDisclaimer";

export default function Home() {
  return (
    <main className="programme landing">
      <header className="masthead">
        <h1 className="masthead-title">osu! Mappool Analyzer</h1>
        <p className="masthead-eyebrow">Tournament mappool analysis</p>
      </header>

      <p className="landing-intro">
        Check your pool before the first pick
      </p>

      <HowToUse />

      <Link href="/tournaments/new" className="btn btn-primary landing-action">
        Analyze a mappool
        <ArrowRight size={16} strokeWidth={1.5} aria-hidden="true" />
      </Link>

      <WipDisclaimer />

      <p className="footer-note">
        Alpha state
        <span className="footer-note-detail">
          This is an early build. Features may change or break.
        </span>
      </p>
    </main>
  );
}
