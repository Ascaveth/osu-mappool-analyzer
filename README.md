<br />
<div align="center">
  <a href="https://github.com/Ascaveth/osu-mappool-analyzer">
    <img src="docs/images/logo.png" alt="Logo" width="480">
  </a>

<h3 align="center">osu! Mappool Analyzer</h3>

  <p align="center">
    This project is a work in progress. Review each analysis suggestion and decide whether it fits your mappool.
    <br />
    <a href="docs/01-vision.md"><strong>Read the docs</strong></a>
    <br />
    <br />
    <a href="https://osu-mappool-analyzer-frontend-production.up.railway.app/">View demo</a>
    &middot;
    <a href="https://github.com/Ascaveth/osu-mappool-analyzer/issues/new?labels=bug">Report a bug</a>
    &middot;
    <a href="https://github.com/Ascaveth/osu-mappool-analyzer/issues/new?labels=enhancement">Request a feature</a>
  </p>
</div>

<details>
  <summary>Table of contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About the project</a>
      <ul>
        <li><a href="#built-with">Built with</a></li>
        <li><a href="#what-the-analyzers-actually-do">What the analyzers actually do</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#layout">Layout</a></li>
    <li><a href="#testing">Testing</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
  </ol>
</details>

## About the project

osu! Mappool Analyzer helps mappoolers and playtesters review tournament pools. It automates checks for balance, composition, diversity, and difficulty progression so they can spend less time checking pools by hand.

- The Go backend in `backend/` runs the analysis engine and exposes tournaments, stages, categories, slots, and beatmaps through a REST API.
- The Next.js frontend in `frontend/` calls the backend through the REST client in `lib/api/`. The tournament creation flow, pool editor, and report view use `app/tournaments/new`, `app/tournaments/[id]/pool`, and `app/tournaments/[id]/report`. The root page at `app/page.tsx` is a standalone demo that uses `lib/sample-data.ts`.

Storage is in memory only. Restarting the backend deletes all tournaments you created. Postgres persistence is planned but has not been implemented.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built with

* [![Go][Go.dev]][Go-url]
* [![Next][Next.js]][Next-url]
* [![React][React.js]][React-url]
* [![TypeScript][TypeScript.dev]][TypeScript-url]
* [![TailwindCSS][Tailwind.com]][Tailwind-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### What the analyzers actually do

The analyzers in `backend/internal/analysis/` check pools at three levels:

- Tournament analyzers in `tournament/` check balance, composition, diversity, and progression across stages. For example, they check whether the Round of 16 is harder than Qualifiers and whether one mod category dominates the pool.

- Metadata analyzers in `metadata/` check BPM range, difficulty settings, mapper repetition, and object density for each beatmap and slot. Their results feed into the tournament analyzers.

- Pattern analyzers in `pattern/` check jump distance, jump angle, slider complexity, spinner usage, and stream bursts. They use beatmap files parsed by `internal/osufile`.

You define the tournament stages, slot counts, and mod categories to match your tournament format.

The Go backend uses the standard library HTTP server without a framework. Its only external dependency is `google/uuid`. It normalizes raw beatmap data before passing it to the analyzers and exposes the results through REST. The Next.js frontend displays those results. All analysis runs on the backend.

The design documents in `docs/`, numbered `01-vision.md` through `16-test-plan.md`, describe the specification. Some of that specification is still awaiting implementation.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- GETTING STARTED -->
## Getting started

### Prerequisites

* Docker and Docker Compose to run both services with one command
* Go 1.21+ and Node.js 18+ to run the backend and frontend manually

### Installation

With Docker:

```sh
docker-compose up
```

The backend runs on `localhost:8080` and the frontend on `localhost:3000`. The frontend build sets `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/v1`. API calls run in the browser, so this address must be reachable from your browser.

Without Docker, run each service in a separate terminal:

```sh
# backend
cd backend && go run ./cmd/server

# frontend
cd frontend && npm install && npm run dev
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE EXAMPLES -->
## Usage

Create a tournament in the frontend and define its stages and mod categories. Add beatmaps in the pool editor, then open the report view to analyze the pool and review findings on composition, progression, balance, and diversity.

### Import a pool from Google Sheets

Open the tab in Google Sheets and copy its URL. It must include a `gid` so the app knows which tab to read. Make sure anyone with the link can view the sheet. This feature requires the backend.

Use one table per stage. Put the stage name one cell to the left of `MOD#` on the row immediately above the table header. Include a `Map ID` column containing numeric osu! beatmap IDs. Extra columns are allowed. Use slot labels such as `NM1` or `HD2`. An unnumbered `FM` works when the stage has exactly one FM slot.

Create the target stages and slots first. In the pool editor, open **Import from Google Sheets**, paste the link, and select **Review import**. Fix any flagged rows and review the tab again. Confirm any replacements before saving. If the batch fails, the pool and beatmap library keep their previous contents.

You can also script these operations through the REST API. Its handlers live in `backend/internal/api`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Layout

```
backend/
  cmd/server           entrypoint
  internal/api         REST handlers, routing, pagination, error responses
  internal/analysis    the engine: tournament/, metadata/, pattern/ analyzers
  internal/config      server configuration loading
  internal/domain      core types: beatmap, difficulty, configuration, analysis results
  internal/enrich      star rating enrichment
  internal/integration pipeline-level integration tests
  internal/modmap      mod combination mapping
  internal/normalize   normalization pipeline
  internal/osuapi      osu! API client + OAuth
  internal/osufile     osu! beatmap file parsing
  internal/report      human-readable report generation
  internal/storage     repository interfaces + in-memory impl

frontend/
  app/tournaments      new-tournament flow, pool editor ([id]/pool), report view ([id]/report)
  app/api/osu-proxy     server route that proxies osu! API calls
  components/          report UI (Masthead, ThesisHero, StageSection, StageNav, MarginNote, Footer, HowToUse, WipDisclaimer)
  components/ui/        shadcn/ui primitives
  lib/api/              REST client that talks to the Go backend
  lib/sample-data.ts    mock data used only by the root demo page
  lib/beatmap-format.ts beatmap display formatting helpers
  lib/types.ts          shared frontend types
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Testing

```sh
cd backend && go test ./...
```

CI runs this command in `.github/workflows/backend-tests.yml`. The frontend has no test suite yet. Its only configured check is `npm run lint`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ROADMAP -->
## Roadmap

- [x] Detect OD progression regressions and spikes across stages ([#2](https://github.com/Ascaveth/osu-mappool-analyzer/pull/2)), star-rating spread issues within stages ([#22](https://github.com/Ascaveth/osu-mappool-analyzer/pull/22)), similar skill profiles ([#28](https://github.com/Ascaveth/osu-mappool-analyzer/pull/28)), and missing skill coverage ([#20](https://github.com/Ascaveth/osu-mappool-analyzer/pull/20)).
- [x] Show findings with reasons and suggested changes ([#2](https://github.com/Ascaveth/osu-mappool-analyzer/pull/2)), and resolve affected locations in reports ([#30](https://github.com/Ascaveth/osu-mappool-analyzer/pull/30)).
- [ ] Persist tournament data so it survives backend restarts. [#41](https://github.com/Ascaveth/osu-mappool-analyzer/issues/41)
- [ ] Show analysis coverage and insufficient-data limits in reports. [#42](https://github.com/Ascaveth/osu-mappool-analyzer/issues/42)
- [ ] Validate analysis results against real tournament pools and feedback from mappoolers. [#43](https://github.com/Ascaveth/osu-mappool-analyzer/issues/43)
- [ ] Import mappools from Google Sheets. [#44](https://github.com/Ascaveth/osu-mappool-analyzer/issues/44)
- [ ] Compare pools across tournaments or editions to identify changes in difficulty and composition. [#45](https://github.com/Ascaveth/osu-mappool-analyzer/issues/45)

See the [open issues](https://github.com/Ascaveth/osu-mappool-analyzer/issues) for proposed features and known gaps.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTRIBUTING -->
## Contributing

### Suggesting something

Open a GitHub issue on [Ascaveth/osu-mappool-analyzer](https://github.com/Ascaveth/osu-mappool-analyzer). For architecture changes or new analyzers, reference the relevant document under `docs/` if one covers the idea.

Proposed changes should help tournament organizers understand or improve their mappools.

### Submitting a change

1. Fork the project
2. Branch off `main`: `feature/<name>`, `bugfix/<name>`, `refactor/<module>`, `docs/<topic>`, etc.
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): summary`, e.g. `feat(analyzer): add mapper diversity check`. Keep one logical change per commit. Submit refactors separately from features.
4. Before opening a PR:
   ```sh
   cd backend && go test ./...
   cd frontend && npm run lint
   ```
5. Open the PR against `main`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Top contributors

<a href="https://github.com/Ascaveth/osu-mappool-analyzer/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=Ascaveth/osu-mappool-analyzer" alt="contrib.rocks image" />
</a>

<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTACT -->
## Contact

Project repository: [https://github.com/Ascaveth/osu-mappool-analyzer](https://github.com/Ascaveth/osu-mappool-analyzer)

Questions or ideas: [open an issue](https://github.com/Ascaveth/osu-mappool-analyzer/issues).

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
[Go.dev]: https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[Next.js]: https://img.shields.io/badge/next.js-000000?style=for-the-badge&logo=nextdotjs&logoColor=white
[Next-url]: https://nextjs.org/
[React.js]: https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB
[React-url]: https://reactjs.org/
[TypeScript.dev]: https://img.shields.io/badge/TypeScript-3178C6?style=for-the-badge&logo=typescript&logoColor=white
[TypeScript-url]: https://www.typescriptlang.org/
[Tailwind.com]: https://img.shields.io/badge/Tailwind_CSS-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white
[Tailwind-url]: https://tailwindcss.com/
