"use client";

import { MoonIcon, SunIcon } from "lucide-react";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";

export default function ThemeSwitch() {
  const { resolvedTheme, systemTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Match the server until the stored theme is available.
    setMounted(true);
  }, []);

  if (!mounted) return null;

  return (
    <label className="flex items-center gap-2">
      <SunIcon size={16} aria-hidden="true" />
      <input
        className="toggle toggle-sm"
        type="checkbox"
        role="switch"
        aria-label="Dark theme"
        checked={resolvedTheme === "dark"}
        onChange={(event) => {
          const target = event.target.checked ? "dark" : "light";
          setTheme(target === systemTheme ? "system" : target);
        }}
      />
      <MoonIcon size={16} aria-hidden="true" />
    </label>
  );
}
