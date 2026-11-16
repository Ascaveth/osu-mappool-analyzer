const STEPS = [
  {
    numeral: "1",
    title: "Set up your tournament",
    body: "Choose stages like Qualifiers or Grand Finals. Set mod categories and slot counts.",
  },
  {
    numeral: "2",
    title: "Add your maps",
    body: "Assign a beatmap to each slot in your pool.",
  },
  {
    numeral: "3",
    title: "Get your report",
    body: "Check difficulty, mod mix, and balance across every stage.",
  },
  {
    numeral: "4",
    title: "Read what it found",
    body: "Read the reason behind each finding. Decide whether it needs a change for your tournament.",
  },
];

/**
 * Renders the landing page's step-by-step guide from tournament setup to a
 * finished analysis report.
 */
export function HowToUse() {
  return (
    <section className="howto" aria-labelledby="howto-title">
      <p className="howto-eyebrow">Getting started</p>
      <h2 className="howto-title" id="howto-title">How to use this analyzer</h2>
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
