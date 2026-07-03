<a id="readme-top"></a>

[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![MIT License][license-shield]][license-url]

<br />
<div align="center">
  <a href="https://github.com/Ascaveth/osu-mappool-analyzer">
    <img src="docs/images/logo.png" alt="Logo" width="480">
  </a>

<h3 align="center">osu! Mappool Analyzer</h3>

  <p align="center">
    This osu! mappool analyzer is a work in progress. Always review each suggestion and decide whether it fits your mappools.
    <br />
    <a href="docs/01-vision.md"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://osu-mappool-analyzer-frontend-production.up.railway.app/">View Demo</a>
    &middot;
    <a href="https://github.com/Ascaveth/osu-mappool-analyzer/issues/new?labels=bug">Report Bug</a>
    &middot;
    <a href="https://github.com/Ascaveth/osu-mappool-analyzer/issues/new?labels=enhancement">Request Feature</a>
  </p>
</div>

<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
        <li><a href="#what-the-analyzers-actually-do">What the analyzers actually do</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
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

## About The Project

Feed it a tournament's stages and beatmaps, and it runs a set of analyzers over the pool. The things a mappooler actually needs to sanity-check before a pool goes live.

This isn't a player stats site, and it's not another leaderboard clone. The mappools itself is the subject.

Two pieces, wired together:

- **Backend** (`backend/`): a Go API with a working analysis engine behind it. Tournaments, stages, categories, slots, beatmaps all modeled and exposed over REST.
- **Frontend** (`frontend/`): a Next.js app. The tournament creation flow, pool editor, and report view (`app/tournaments/new`, `app/tournaments/[id]/pool`, `app/tournaments/[id]/report`) call the backend through a real REST client in `lib/api/`. The root page (`app/page.tsx`) is a standalone demo that still renders off `lib/sample-data.ts`.

Storage is in-memory only. Restart the backend and every tournament you created is gone; there's no Postgres or any other persistence layer plugged in yet, despite that being the long-term plan.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

* [![Go][Go.dev]][Go-url]
* [![Next][Next.js]][Next-url]
* [![React][React.js]][React-url]
* [![TypeScript][TypeScript.dev]][TypeScript-url]
* [![TailwindCSS][Tailwind.com]][Tailwind-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### What the analyzers actually do

Under `backend/internal/analysis/`:

**Tournament-level** (`tournament/`): balance, composition, diversity, progression. These look at a pool across its whole stage list: is Round of 16 meaningfully harder than Qualifiers, does one mod category dominate, that sort of question.

**Metadata-level** (`metadata/`): BPM range, difficulty settings, mapper repetition, object density. Per-beatmap and per-slot checks that feed into the tournament-level analyzers above.

**Pattern-level** (`pattern/`): jump distance, jump angle, slider complexity, spinner usage, stream bursts. Lower-level readouts of what's actually in the beatmap file, parsed by `internal/osufile`.

None of this is hardcoded to a fixed bracket shape. Tournament structure stages, slot counts, mod categories is user-defined, because no two tournaments run the same format.

Go backend, stdlib HTTP plus `google/uuid` (that's the only external dependency, no framework). It exposes a REST API over the analysis engine and normalizes raw beatmap data before anything touches an analyzer. The Next.js frontend is a presentation layer on top; it doesn't do analysis itself.

Full design docs live in `docs/` (`01-vision.md` through `16-test-plan.md`). Treat those as specification, not a changelog of what's built — plenty of it is still ahead of the code.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- GETTING STARTED -->
## Getting Started

### Prerequisites

* Docker and Docker Compose, if you want the one-command path
* Go 1.21+ and Node.js 18+, if you'd rather run each side manually

### Installation

**With Docker:**

```sh
docker-compose up
```

Backend on `localhost:8080`, frontend on `localhost:3000`. The frontend build bakes in `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/v1` at build time since the API calls happen client-side, not from Next's server.

**Without Docker:**

```sh
# backend
cd backend && go run ./cmd/server

# frontend
cd frontend && npm install && npm run dev
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE EXAMPLES -->
## Usage

Create a tournament through the frontend's new-tournament flow, define its stages and mod categories, then drop beatmaps into the pool editor. The report view runs the analysis engine over the pool and renders composition, progression, balance, and diversity findings.

Everything the frontend shows is also available directly over the REST API in `backend/internal/api` if you want to script against it.

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

That's what CI runs (`.github/workflows/backend-tests.yml`). The frontend has no test suite yet — `npm run lint` is the only check configured there.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ROADMAP -->
## Roadmap

- [ ] Postgres persistence layer (storage is in-memory only right now)
- [ ] Frontend test suite
- [ ] Additional analyzers beyond the current tournament/metadata/pattern set

See the [open issues](https://github.com/Ascaveth/osu-mappool-analyzer/issues) for the full list of proposed features and known gaps.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTRIBUTING -->
## Contributing

### Suggesting something

Open a GitHub issue on [Ascaveth/osu-mappool-analyzer](https://github.com/Ascaveth/osu-mappool-analyzer). If the idea touches architecture or adds a new analyzer, point at the relevant file under `docs/` if one already covers it.

If it doesn't produce or improve an insight for tournament organizers, it's probably out of scope.

### Submitting a change

1. Fork the project
2. Branch off `main`: `feature/<name>`, `bugfix/<name>`, `refactor/<module>`, `docs/<topic>`, etc.
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): summary`, e.g. `feat(analyzer): add mapper diversity check`. One logical change per commit — don't mix a refactor into a feature commit.
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

Project Link: [https://github.com/Ascaveth/osu-mappool-analyzer](https://github.com/Ascaveth/osu-mappool-analyzer)

Questions or ideas: [open an issue](https://github.com/Ascaveth/osu-mappool-analyzer/issues).

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
[contributors-shield]: https://img.shields.io/github/contributors/Ascaveth/osu-mappool-analyzer.svg?style=for-the-badge
[contributors-url]: https://github.com/Ascaveth/osu-mappool-analyzer/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/Ascaveth/osu-mappool-analyzer.svg?style=for-the-badge
[forks-url]: https://github.com/Ascaveth/osu-mappool-analyzer/network/members
[stars-shield]: https://img.shields.io/github/stars/Ascaveth/osu-mappool-analyzer.svg?style=for-the-badge
[stars-url]: https://github.com/Ascaveth/osu-mappool-analyzer/stargazers
[issues-shield]: https://img.shields.io/github/issues/Ascaveth/osu-mappool-analyzer.svg?style=for-the-badge
[issues-url]: https://github.com/Ascaveth/osu-mappool-analyzer/issues
[license-shield]: https://img.shields.io/github/license/Ascaveth/osu-mappool-analyzer.svg?style=for-the-badge
[license-url]: https://github.com/Ascaveth/osu-mappool-analyzer/blob/main/LICENSE
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
