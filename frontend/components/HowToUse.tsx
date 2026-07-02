const STEPS = [
  {
    numeral: "1",
    title: "Set up your tournament",
    body: "Add your stages (like Qualifiers or Grand Finals), pick your mod categories, and say how many maps go in each. You can match any bracket format.",
  },
  {
    numeral: "2",
    title: "Add your maps",
    body: "Fill in each slot with the beatmap you've picked for it.",
  },
  {
    numeral: "3",
    title: "Get your report",
    body: "The analyzer checks your pool's difficulty, mod mix, and balance across every stage.",
  },
  {
    numeral: "4",
    title: "Read what it found",
    body: "Each note explains why something might be an issue, not just that it is one — so you can decide if it's worth fixing.",
  },
];

/**
 * Renders the landing page's step-by-step guide from tournament setup to a
 * finished analysis report.
 */
export function HowToUse() {
  return (
    <section className="howto reveal" style={{ animationDelay: "160ms" }}>
      <p className="howto-eyebrow">Getting started</p>
      <h2 className="howto-title">How to use this analyzer</h2>
      <ol className="howto-steps">
        {STEPS.map((step) => (
          <li className="howto-step" key={step.numeral}>
            <span className="howto-step-num" aria-hidden="true">
              {step.numeral}
            </span>
            <div className="howto-step-body">
              <h3 className="howto-step-title">{step.title}</h3>
              <p className="howto-step-desc">{step.body}</p>
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}
