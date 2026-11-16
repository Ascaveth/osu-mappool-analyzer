# Vision

## What this project is

osu! Mappool Analyzer is an analysis platform for osu! tournament mappools. It evaluates the quality, balance, diversity, and progression of a mappool and produces structured, human-readable insight for the people who build and review pools.

## What this project is not

- Not a player or team statistics tool.
- Not a match predictor or scoring tool.
- Not a tournament management or scheduling system.
- Not a visualization tool that happens to do some analysis. Visualization is an output, not the product.

## The core idea

A mappool is a set of beatmaps arranged into stages and mod categories to test players fairly, with increasing difficulty across a tournament. Pool quality is currently judged by hosts' intuition and community feedback after the fact. This project gives that judgment a structured, repeatable, explainable basis: before and after a pool is released.

Each feature should answer this question:

> What insight does this provide to a tournament organizer or mappooler?

## Primary users

- Mappoolers: deciding which maps go into a pool, and whether the set as a whole is balanced and progressive.
- Tournament organizers / hosts: reviewing a finished pool for quality issues before it ships to players.

## Target outcome

Given a tournament's configuration and its beatmaps, the system produces:

- Quantified metrics per stage/category/pool (difficulty, BPM, length, mapper/song diversity, pattern characteristics, etc.)
- Detected issues (difficulty spikes, weak progression, repeated map styles, missing skill coverage) with severity and reasoning.
- A human-readable report explaining what was found and why it matters, with concrete recommendations.

## Why analysis-first

The Analysis Engine produces the findings that charts and dashboards present. The API, reports, and UI are replaceable presentation layers over its output. See [Architecture Principles](04-architecture-principles.md).
