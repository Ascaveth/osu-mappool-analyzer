import { NextRequest, NextResponse } from "next/server";

export async function GET(req: NextRequest) {
  const id = req.nextUrl.searchParams.get("id");
  if (!id || !/^\d+$/.test(id)) {
    return new NextResponse("Missing or invalid beatmap id", { status: 400 });
  }

  let res: Response;
  try {
    res = await fetch(`https://osu.ppy.sh/osu/${id}`, {
      headers: { "User-Agent": "osu-mappool-analyzer/1.0" },
      signal: AbortSignal.timeout(10_000),
    });
  } catch (e) {
    const msg = e instanceof Error && e.name === "TimeoutError" ? "timed out" : `${e}`;
    return new NextResponse(`Failed to reach osu!: ${msg}`, { status: 502 });
  }

  if (!res.ok) {
    return new NextResponse(`osu! returned ${res.status}`, { status: 502 });
  }

  const text = await res.text();
  return new NextResponse(text, {
    status: 200,
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
