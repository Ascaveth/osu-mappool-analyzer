# osu! Mappool Analyzer

<p>Analyze osu! tournament mappools with automated insights for balance, progression, diversity, and overall quality.</p>

<p align="center">

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go)
![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=nextdotjs)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)

</p>

---

## Features

| Feature | Description |
|---------|-------------|
| Composition Analysis | Evaluate mod distribution and stage composition |
| Progression Analysis | Detect difficulty spikes and progression issues |
| Diversity Analysis | Measure mapper, artist, and song variety |
| Balance Analysis | Check consistency across the entire pool |
| Validation Engine | Identify common mappool design issues |
| Report Generation | Produce structured reports |

---

## Quick Start

### Requirements

- Go 1.21+
- Node.js 20+
- PostgreSQL

```bash
git clone https://github.com/Ascaveth/osu-mappool-analyzer.git
cd osu-mappool-analyzer

cd backend
go mod download
go run ./cmd/main.go

cd ../frontend
npm install
npm run dev
```

---

## Architecture

```text
Tournament
    │
    ▼
Normalization
    │
    ▼
Analysis Engine
    ├── Composition
    ├── Progression
    ├── Diversity
    ├── Balance
    └── Validation
    │
    ▼
REST API / Web UI / Reports
```

---

## Project Structure

```text
backend/
frontend/
docs/
```

---

## Documentation

See the `docs/` directory for architecture, API, domain model, UI, and testing documentation.

---

## Roadmap

- [x] Beatmap parser
- [x] Analysis engine
- [x] REST API
- [ ] Advanced analyzers
- [ ] Multi-pool comparison
- [ ] Historical analysis

---

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests
4. Commit using Conventional Commits
5. Open a Pull Request

---

## License

MIT

---
