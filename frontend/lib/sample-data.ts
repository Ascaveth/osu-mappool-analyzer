import type { Tournament, Report } from "./types";

// Sample tournament data for the flagship Report view (docs/15-ui-specification.md).
// All song/artist/mapper names are invented for this mockup — no real
// osu! community members or tracks are referenced. The pool is
// deliberately imperfect: a balanced Qualifiers stage, a Round of 16
// that over-relies on one mod category, and a Grand Finals with a
// difficulty cooldown and a repeated song — exactly the kind of issues
/**
 * Builds a beatmap object for the mockup, mirroring the actual API shape
 * the viewer consumes.
 *
 * @param id - Beatmap identifier
 * @param title - Beatmap title
 * @param artist - Beatmap artist
 * @param mapper - Beatmap creator
 * @param version - Difficulty name (osu!'s "Version" field), e.g. "Insane"
 * @param ar - Approach rate
 * @param od - Overall difficulty
 * @param bpm - Beatmap tempo
 * @returns A beatmap object containing the provided fields
 */
function bm(
  id: string,
  title: string,
  artist: string,
  mapper: string,
  version: string,
  ar: number,
  od: number,
  bpm: number,
) {
  return { id, title, artist, mapper, version, ar, od, bpm };
}

export const tournament: Tournament = {
  id: "t-spring-invitational",
  name: "Spring Invitational",
  edition: "2026",
  stages: [
    {
      id: "stage-qualifiers",
      name: "Qualifiers",
      order: 1,
      categories: [
        {
          id: "cat-q-nm",
          name: "NM",
          order: 1,
          slots: [
            { id: "s1", code: "NM1", beatmap: bm("b1", "Glasswing", "Aurel Sky", "rotodrift", "Insane", 8.6, 7.4, 178) },
            { id: "s2", code: "NM2", beatmap: bm("b2", "Tidesong", "Marrow Veil", "Quillfeather", "Insane", 8.8, 7.6, 182) },
            { id: "s3", code: "NM3", beatmap: bm("b3", "Cinder & Salt", "Hollow Pines", "Lacewing", "Extra", 9.0, 7.8, 190) },
            { id: "s4", code: "NM4", beatmap: bm("b4", "Paper Lanterns", "Iyo Naka", "rotodrift", "Extra", 9.0, 8.0, 196) },
            { id: "s5", code: "NM5", beatmap: bm("b5", "Static Bloom", "Verdance", "Mireille", "Extreme", 9.1, 8.1, 202) },
          ],
        },
        {
          id: "cat-q-hd",
          name: "HD",
          order: 2,
          slots: [
            { id: "s6", code: "HD1", beatmap: bm("b6", "Vertigo Hour", "Kessen", "Lacewing", "Hyper", 9.2, 8.2, 188) },
            { id: "s7", code: "HD2", beatmap: bm("b7", "Low Tide Choir", "Marrow Veil", "Velin", "Hyper", 9.3, 8.3, 196) },
          ],
        },
        {
          id: "cat-q-hr",
          name: "HR",
          order: 3,
          slots: [
            { id: "s8", code: "HR1", beatmap: bm("b8", "Brittle Glass", "Aurel Sky", "Quillfeather", "Insane", 9.4, 8.4, 184) },
            { id: "s9", code: "HR2", beatmap: bm("b9", "Iron Orchard", "Hollow Pines", "Mireille", "Insane", 9.5, 8.6, 192) },
          ],
        },
        {
          id: "cat-q-dt",
          name: "DT",
          order: 4,
          slots: [
            { id: "s10", code: "DT1", beatmap: bm("b10", "Filament", "Iyo Naka", "Velin", "Extra", 9.0, 8.0, 174) },
            { id: "s11", code: "DT2", beatmap: bm("b11", "Quiet Static", "Verdance", "Lacewing", "Extra", 9.1, 8.2, 180) },
          ],
        },
        {
          id: "cat-q-fm",
          name: "FM",
          order: 5,
          slots: [
            { id: "s12", code: "FM1", beatmap: bm("b12", "Driftwood Atlas", "Kessen", "Mireille", "Insane", 8.7, 7.6, 168) },
            { id: "s13", code: "FM2", beatmap: bm("b13", "Halflight", "Hollow Pines", "rotodrift", "Extra", 9.2, 8.0, 200) },
          ],
        },
      ],
    },
    {
      id: "stage-ro16",
      name: "Round of 16",
      order: 2,
      categories: [
        {
          id: "cat-r-nm",
          name: "NM",
          order: 1,
          slots: [
            { id: "s14", code: "NM1", beatmap: bm("b14", "Threadbare", "Aurel Sky", "Doverhall", "Insane", 8.4, 6.8, 172) },
            { id: "s15", code: "NM2", beatmap: bm("b15", "Slow Static", "Marrow Veil", "Doverhall", "Insane", 8.5, 6.9, 176) },
            { id: "s16", code: "NM3", beatmap: bm("b16", "Ashen Field", "Verdance", "Doverhall", "Insane", 8.6, 7.0, 180) },
            { id: "s17", code: "NM4", beatmap: bm("b17", "Greywater", "Hollow Pines", "Doverhall", "Insane", 8.6, 7.0, 182) },
            { id: "s18", code: "NM5", beatmap: bm("b18", "Coldframe", "Kessen", "Doverhall", "Extra", 8.7, 7.1, 186) },
            { id: "s19", code: "NM6", beatmap: bm("b19", "Undertow", "Iyo Naka", "Doverhall", "Extra", 8.7, 7.2, 190) },
          ],
        },
        {
          id: "cat-r-hd",
          name: "HD",
          order: 2,
          slots: [{ id: "s20", code: "HD1", beatmap: bm("b20", "Needle's Eye", "Marrow Veil", "Velin", "Hyper", 9.2, 7.6, 196) }],
        },
        {
          id: "cat-r-fm",
          name: "FM",
          order: 3,
          slots: [{ id: "s21", code: "FM1", beatmap: bm("b21", "Wax & Wane", "Aurel Sky", "Mireille", "Insane", 9.0, 7.4, 178) }],
        },
      ],
    },
    {
      id: "stage-finals",
      name: "Grand Finals",
      order: 3,
      categories: [
        {
          id: "cat-f-nm",
          name: "NM",
          order: 1,
          slots: [
            { id: "s22", code: "NM1", beatmap: bm("b22", "Last Light", "Hollow Pines", "Lacewing", "Extreme", 9.4, 8.6, 188) },
            { id: "s23", code: "NM2", beatmap: bm("b23", "Carrion Bloom", "Verdance", "Quillfeather", "Extreme", 9.5, 8.8, 192) },
            { id: "s24", code: "NM3", beatmap: bm("b24", "Hollow Crown", "Kessen", "rotodrift", "Apex", 9.6, 8.9, 196) },
          ],
        },
        {
          id: "cat-f-hd",
          name: "HD",
          order: 2,
          slots: [
            { id: "s25", code: "HD1", beatmap: bm("b25", "Briar & Bone", "Marrow Veil", "Mireille", "Apex", 9.6, 9.0, 200) },
          ],
        },
        {
          id: "cat-f-hr",
          name: "HR",
          order: 3,
          slots: [
            { id: "s26", code: "HR1", beatmap: bm("b26", "Wrought Iron", "Aurel Sky", "Velin", "Extreme", 9.7, 9.0, 198) },
            { id: "s27", code: "HR2", beatmap: bm("b27", "Stormglass", "Iyo Naka", "Lacewing", "Extreme", 9.7, 9.0, 204) },
          ],
        },
        {
          id: "cat-f-fm",
          name: "FM",
          order: 4,
          slots: [
            { id: "s28", code: "FM1", beatmap: bm("b28", "Last Light", "Hollow Pines", "Lacewing", "Extreme", 9.4, 8.6, 188) },
          ],
        },
      ],
    },
  ],
};

export const report: Report = {
  scope: { type: "tournament", id: tournament.id },
  generatedAt: "2026-07-01T09:00:00Z",
  sections: {
    summary:
      "Difficulty cools off right when Round of 16 should be raising the stakes — its average Overall Difficulty drops below Qualifiers, and three quarters of its slots lean on the same Normal Mod category. Grand Finals recovers the climb but reuses “Last Light” across two categories, and its Hard Rock pair gives players no approach-rate range to read.",
    findings: [
      {
        analyzerName: "balance-analyzer",
        scope: { type: "stage", id: "stage-ro16" },
        finding: {
          severity: "warning",
          description: "Free Mod maps is more favoured into Hidden rather than Hard Rock.",
          reason: "Looking at the maps, FM1 and FM2 is Hidden favoured meanwhile only FM3 is Hard Rock favoured",
          recommendation: "rebalance slot counts across categories, or reconsider whether this stage needs that many categories",
        },
      },
      {
        analyzerName: "length",
        scope: { type: "tournament", id: tournament.id },
        finding: {
          severity: "warning",
          description: "Average drain time for the whole pool is more than 5 minutes.",
          reason: "Average drain time for the whole pool this stage is too long, might reduce the drain time into average 2:30",
          recommendation: "review beatmap selection in “Round of 16” relative to “Qualifiers”, or confirm the difficulty decrease is intentional for this tournament format",
          targetStageId: "stage-ro16",
        },
      },
      {
        analyzerName: "balance-analyzer",
        scope: { type: "category", id: "cat-f-hr" },
        finding: {
          severity: "warning",
          description: "every beatmap in this category shares the same AR (9.7)",
          reason: "zero variation on an axis gives players no range to adapt across within the category, regardless of how it compares to other categories",
          recommendation: "vary AR across this category's slots, or confirm the uniform value is an intentional category rule",
        },
      },
      {
        analyzerName: "diversity-analyzer",
        scope: { type: "stage", id: "stage-finals" },
        finding: {
          severity: "warning",
          description: "“Last Light” by Hollow Pines appears in more than one slot within this stage",
          reason: "the same song in two categories tests overlapping musical familiarity rather than two genuinely different pool slots",
          recommendation: "replace one occurrence with a distinct song",
        },
      },
    ],
    warnings: [],
    recommendations: [
      "rebalance slot counts across categories, or reconsider whether this stage needs that many categories",
      "review beatmap selection in “Round of 16” relative to “Qualifiers”, or confirm the difficulty decrease is intentional for this tournament format",
      "vary AR across this category's slots, or confirm the uniform value is an intentional category rule",
      "replace one occurrence with a distinct song",
    ],
    statistics: {
      total_analyses: 12,
      total_findings: 4,
      findings_warning: 4,
      findings_critical: 0,
    },
  },
};

report.sections.warnings = [...report.sections.findings];
