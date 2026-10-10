import { useEffect, useState } from "react";

export type Theme = "light" | "dark";
const KEY = "theme";

function stored(): Theme | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === "light" || v === "dark" ? v : null;
  } catch {
    return null;
  }
}

function system(): Theme {
  return typeof matchMedia === "function" &&
    matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

// Unset data-theme means "follow the OS"; a stored choice overrides it.
export function useTheme() {
  const [choice, setChoice] = useState<Theme | null>(stored);
  const theme = choice ?? system();

  useEffect(() => {
    const root = document.documentElement;
    if (choice) root.dataset.theme = choice;
    else delete root.dataset.theme;
  }, [choice]);

  function setTheme(t: Theme) {
    try {
      localStorage.setItem(KEY, t);
    } catch {
      // ponytail: private mode — the choice lasts for this tab only.
    }
    setChoice(t);
  }

  return {
    theme,
    setTheme,
    toggle: () => setTheme(theme === "dark" ? "light" : "dark"),
  };
}
