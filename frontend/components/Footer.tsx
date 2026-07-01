import Link from "next/link";

/**
 * Sitewide footer, styled as the colophon page of a printed programme.
 */
export function Footer() {
  return (
    <footer className="site-footer">
      <div className="footer-grid">
        <div>
          <p className="footer-brand">osu! Mappool Analyzer</p>
          <p className="footer-tagline">
            Reads a mappool the way a seasoned mappooler would, and writes down what it finds.
          </p>
        </div>

        <div>
          <p className="footer-heading">Navigate</p>
          <nav className="footer-links" aria-label="Footer">
            <Link href="/">Home</Link>
            <Link href="/tournaments/new">New Tournament</Link>
          </nav>
        </div>

        <div>
          <p className="footer-heading">Colophon</p>
          <ul className="footer-links">
            <li>
              <a href="https://github.com/Ascaveth/osu-mappool-analyzer" target="_blank" rel="noopener noreferrer">
                Source on GitHub
              </a>
            </li>
            <li className="footer-meta-line">Demo build · analysis engine v0.1</li>
          </ul>
        </div>
      </div>

      <p className="footer-fineprint">© {new Date().getFullYear()} osu! Mappool Analyzer — not affiliated with osu!</p>
    </footer>
  );
}
