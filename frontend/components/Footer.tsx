import Link from "next/link";
import { SITE_TAGLINE } from "@/lib/site-metadata";

/**
 * Sitewide footer with site navigation and project info.
 */
export function Footer() {
  return (
    <footer className="site-footer">
      <div className="footer-grid">
        <div>
          <p className="footer-brand">osu! Mappool Analyzer</p>
          <p className="footer-tagline">{SITE_TAGLINE}</p>
        </div>

        <div>
          <p className="footer-heading">Navigate</p>
          <nav className="footer-links" aria-label="Footer">
            <Link href="/">Home</Link>
            <Link href="/tournaments/new">New Tournament</Link>
          </nav>
        </div>

        <div>
          <p className="footer-heading">About</p>
          <ul className="footer-links">
            <li>
              <a href="https://github.com/Ascaveth/osu-mappool-analyzer" target="_blank" rel="noopener noreferrer">
                Source on GitHub
              </a>
            </li>
            <li className="footer-meta-line">Alpha Build - v1.00</li>
          </ul>
        </div>
      </div>

      <p className="footer-fineprint">© {new Date().getFullYear()} osu! Mappool Analyzer — not affiliated with osu!</p>
    </footer>
  );
}
