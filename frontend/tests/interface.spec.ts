import { expect, test, type Page, type Locator } from "@playwright/test";
import type { Beatmap, Tournament } from "../lib/types";

const beatmap: Beatmap = {
  id: "review-map",
  title: "A long beatmap title with enough words to exercise narrow ledger rows",
  artist: "Review artist",
  mapper: "Review mapper",
  version: "An extended difficulty name",
  ar: 9.6, od: 8, cs: 3.5, hp: 4, bpm: 180,
  coverUrl: "/review-cover.svg",
};

async function openPool(page: Page, filled = false) {
  await page.goto("/");
  await page.route("**/review-cover.svg", (route) => route.fulfill({
    contentType: "image/svg+xml",
    body: '<svg xmlns="http://www.w3.org/2000/svg" width="800" height="200"><rect width="800" height="200" fill="#6e86a0"/></svg>',
  }));
  const tournament: Tournament = {
    id: "review", name: "Interface Review Cup", edition: "",
    stages: ["Qualifiers", "Grand Finals"].map((name, index) => ({
      id: `stage-${index}`, name, order: index + 1,
      projectedStarRating: 6.75,
      categories: [{
        id: `category-${index}`, name: "No Mod", order: 1,
        slots: [1, 2].map((number) => ({
          id: `slot-${index}-${number}`, code: `NM${number}`, beatmap: null,
        })),
      }],
    })),
  };
  await page.evaluate(({ tournament, beatmap, filled }) => {
    localStorage.setItem("osu-analyzer-mock", JSON.stringify({
      tournaments: { [tournament.id]: tournament },
      beatmaps: { [beatmap.id]: beatmap },
      slotAssignments: filled ? Object.fromEntries(tournament.stages.flatMap((stage) =>
        stage.categories.flatMap((category) => category.slots.map((slot) => [slot.id, beatmap.id])))) : {},
    }));
  }, { tournament, beatmap, filled });
  await page.goto("/tournaments/review/pool");
  await expect(page.getByRole("heading", { name: "Import Beatmaps" })).toBeVisible();
}

async function expectRowsToFit(page: Page) {
  expect(await page.locator(".slot-row").evaluateAll((rows) => rows.flatMap((row) => {
    const bounds = row.getBoundingClientRect();
    return [...row.children].filter((child) => {
      const rect = child.getBoundingClientRect();
      return rect.left < bounds.left - 1 || rect.right > bounds.right + 1;
    }).map((child) => child.textContent);
  }))).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
}

test("stage names and input text remain usable at 320px", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await page.goto("/tournaments/new");
  await page.getByRole("button", { name: "+ Add Stage", exact: true }).click();
  const names = page.getByLabel("Stage name", { exact: true });
  await names.nth(0).fill("Qualifiers");
  await names.nth(1).fill("Grand Finals");
  for (const input of await names.all()) {
    expect((await input.boundingBox())!.width).toBeGreaterThan(200);
  }
  const sizes = await page.locator("input.input").evaluateAll((inputs) =>
    inputs.map((input) => Number.parseFloat(getComputedStyle(input).fontSize)));
  expect(sizes.every((size) => size >= 16)).toBe(true);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath("setup-320.png"), fullPage: true, animations: "disabled" });
});

test("filled pool and report rows reflow in both themes", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await openPool(page, true);
  for (const dark of [false, true]) {
    await page.getByRole("switch", { name: "Dark theme" }).setChecked(dark);
    for (const width of [320, 640, 861, 1280]) {
      await page.setViewportSize({ width, height: 800 });
      await expectRowsToFit(page);
    }
  }
  await page.setViewportSize({ width: 320, height: 800 });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath("pool-320.png"), fullPage: true, animations: "disabled" });
  await page.getByRole("button", { name: "Run Analysis" }).click();
  await expect(page.getByText("Analysis results:", { exact: false })).toBeVisible();
  await expect(page.locator(".stage").last()).toHaveCSS("opacity", "1");
  for (const dark of [false, true]) {
    await page.getByRole("switch", { name: "Dark theme" }).setChecked(dark);
    for (const width of [320, 640, 861, 1280]) {
      await page.setViewportSize({ width, height: 800 });
      await expectRowsToFit(page);
    }
  }
  await page.setViewportSize({ width: 320, height: 800 });
  await expect(page.getByText("High mapper repetition reduces pool diversity.")).toBeVisible();
  await expect(page.getByText("Replace some maps by Review mapper with alternatives from other mappers.")).toBeVisible();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: testInfo.outputPath("report-320.png"), fullPage: true, animations: "disabled" });
});

test("pending import blocks background focus and restores the failed action", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await openPool(page);
  let release!: () => void;
  const held = new Promise<void>((resolve) => { release = resolve; });
  await page.route("**/api/osu-proxy*", async (route) => {
    await held;
    await route.fulfill({ status: 503, body: "Provider unavailable" });
  });
  const slot = page.getByLabel("Beatmap URL or ID for slot NM1", { exact: true }).first();
  const action = page.getByRole("button", { name: "Import and assign beatmap to slot NM1", exact: true }).first();
  await slot.fill("3176982");
  await action.click();
  await expect(page.getByRole("status")).toContainText("Importing beatmap");
  await expect(page.getByRole("dialog")).toBeFocused();
  await expect(page.locator(".loading-overlay .loading")).toHaveCSS("mask-image", "none");
  await page.keyboard.press("Tab");
  expect(await page.evaluate(() => {
    const dialog = document.querySelector("dialog");
    return document.activeElement === document.body || dialog?.contains(document.activeElement);
  })).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toBeVisible();
  release();
  await expect(page.locator("main").getByRole("alert")).toContainText("check your connection, and import again");
  await expect(slot).toHaveAttribute("aria-invalid", "true");
  await expect(slot).toHaveAccessibleDescription(/Use a beatmap ID such as 3176982/);
  await expect(action).toBeFocused();
});

test("invalid beatmap input gives recovery instructions without a provider request", async ({ page }) => {
  await openPool(page);
  const input = page.getByLabel("Beatmap URL or ID for slot NM1", { exact: true }).first();
  await input.fill("not-a-beatmap");
  await input.press("Enter");
  await expect(input).toHaveAccessibleDescription(/Use a beatmap ID such as 3176982/);
  await expect(page.locator("main").getByRole("alert")).toContainText("import again");
});

test("bulk import fills the pool and enables analysis", async ({ page }) => {
  await openPool(page);
  await page.route("**/api/osu-proxy*", (route) => route.fulfill({
    body: "osu file format v14\n[Metadata]\nTitle:Fixture\nArtist:Artist\nCreator:Mapper\nVersion:Hard\n[Difficulty]\nApproachRate:9\nOverallDifficulty:8\nCircleSize:4\nHPDrainRate:5\n",
  }));
  for (const input of await page.locator(".slot-input").all()) await input.fill("3176982");
  await page.getByRole("button", { name: "Import All (4)", exact: true }).click();
  await expect(page.getByText("4 / 4 slots filled")).toBeVisible();
  await expect(page.getByRole("button", { name: "Run Analysis" })).toBeEnabled();
  await expect(page.getByRole("heading", { name: "Import Beatmaps" })).toBeFocused();
});

async function contrastRatio(element: Locator, foreground: "color" | "borderTopColor" = "color") {
  return element.evaluate((element, foreground) => {
    const style = getComputedStyle(element);
    const rgb = (color: string) => {
      const canvas = document.createElement("canvas");
      const context = canvas.getContext("2d")!;
      context.fillStyle = color;
      context.fillRect(0, 0, 1, 1);
      return [...context.getImageData(0, 0, 1, 1).data].slice(0, 3);
    };
    const luminance = (color: string) => rgb(color).map((channel) => {
      const value = channel / 255;
      return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
    }).reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0);
    const a = luminance(style[foreground]), b = luminance(style.backgroundColor);
    return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
  }, foreground);
}

test("primary hover text passes contrast in both themes", async ({ page }) => {
  await page.goto("/");
  for (const dark of [false, true]) {
    await page.getByRole("switch", { name: "Dark theme" }).setChecked(dark);
    const action = page.getByRole("link", { name: "Analyze a Mappool" });
    await action.hover();
    await expect.poll(() => contrastRatio(action)).toBeGreaterThanOrEqual(4.5);
  }
});

test("input boundaries pass contrast in both themes", async ({ page }) => {
  await page.goto("/tournaments/new");
  for (const dark of [false, true]) {
    await page.getByRole("switch", { name: "Dark theme" }).setChecked(dark);
    await expect.poll(() => contrastRatio(page.getByLabel("Tournament Name"), "borderTopColor"))
      .toBeGreaterThanOrEqual(3);
  }
});

test("reduced motion keeps theme switching static and persists the selection", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  const toggle = page.getByRole("switch", { name: "Dark theme" });
  await toggle.setChecked(true);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  expect(await toggle.evaluate((element) => getComputedStyle(element, "::before").transitionDuration)).toBe("0s");
  await page.reload();
  await expect(toggle).toBeChecked();
  await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
  await toggle.setChecked(false);
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(toggle).toBeChecked();
});

test("ledger content survives 200% text resize", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await openPool(page, true);
  await page.evaluate(() => { document.documentElement.style.fontSize = "200%"; });
  await expectRowsToFit(page);
  await page.getByRole("button", { name: "Run Analysis" }).click();
  await expect(page.getByText("Analysis results:", { exact: false })).toBeVisible();
  await expectRowsToFit(page);
});

test("setup creates the configured tournament through the migrated controls", async ({ page }) => {
  await page.goto("/tournaments/new");
  await page.getByLabel("Tournament Name").fill("Configured Cup");
  await page.getByLabel("Stage name", { exact: true }).fill("Qualifiers");
  await page.getByLabel("Projected SR (optional)").fill("6,75");
  await page.getByRole("button", { name: "Continue to Pool" }).click();
  await expect(page.getByRole("heading", { name: "Qualifiers" })).toBeVisible();
  await expect(page.getByText("★ 6.75 projected")).toBeVisible();
  await expect(page.getByText("0 / 2 slots filled", { exact: false })).toBeVisible();
});

test("stage bulk paste retains preview, import locking, and focus restoration", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 800 });
  await openPool(page);
  const input = page.getByLabel("Bulk paste beatmap IDs", { exact: false }).first();
  await input.fill("3176982");
  await page.getByRole("button", { name: "Preview mapping", exact: true }).first().click();
  await expect(page.locator("main").getByRole("alert")).toContainText("Expected 2 IDs");
  await expect(page.getByText("0 / 4 slots filled", { exact: false })).toBeVisible();
  await input.fill("3176982\nhttps://osu.ppy.sh/beatmaps/3176983");
  await page.getByRole("button", { name: "Preview mapping", exact: true }).first().click();
  await expect(page.getByText("Confirm mapping (2 slots)")).toBeVisible();
  let release!: () => void;
  const held = new Promise<void>((resolve) => { release = resolve; });
  await page.route("**/api/osu-proxy*", async (route) => {
    await held;
    await route.fulfill({ body: "osu file format v14\n[Metadata]\nTitle:Bulk fixture\nArtist:Artist\nCreator:Mapper\nVersion:Hard\n[Difficulty]\nApproachRate:9\nOverallDifficulty:8\nCircleSize:4\nHPDrainRate:5\n" });
  });
  await page.getByRole("button", { name: "Apply 2 to Qualifiers", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeFocused();
  await expect(input).toBeDisabled();
  release();
  await expect(page.getByText("2 / 4 slots filled", { exact: false })).toBeVisible();
  await expect(input).toHaveValue("");
  await expect(input).toBeFocused();
  await expect(page.getByRole("button", { name: "Run Analysis" })).toBeDisabled();
  await expectRowsToFit(page);
});
