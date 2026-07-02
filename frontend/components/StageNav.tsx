import type { Stage } from "@/lib/types";

/**
 * Sticky in-page index for jumping between stages in a report.
 *
 * Only worth rendering once a pool has enough stages that scrolling
 * between them stops being trivial.
 */
export function StageNav({ stages }: { stages: Stage[] }) {
  if (stages.length <= 2) return null;

  return (
    <nav className="stage-nav" aria-label="Jump to stage">
      {stages.map((stage) => (
        <a key={stage.id} className="stage-nav-link" href={`#stage-${stage.id}`}>
          {stage.name}
        </a>
      ))}
    </nav>
  );
}
