import { useEffect, useState, type ReactNode, } from "react";
import { ThemeContext, } from "@/components/theme-context";

type Theme = "light" | "dark";

export function ThemeProvider({ children, defaultTheme = "light", }: { children: ReactNode; defaultTheme?: Theme },) {
  const [theme, setThemeState,] = useState<Theme>(() => {
    try {
      const stored = localStorage.getItem("theme",);
      if (stored === "light" || stored === "dark") return stored;
    } catch {
      /* empty */
    }
    return defaultTheme;
  },);

  const setTheme = (t: Theme,) => {
    setThemeState(t,);
    try {
      localStorage.setItem("theme", t,);
    } catch {
      /* empty */
    }
  };

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark",);
  }, [theme,],);

  return <ThemeContext.Provider value={{ theme, setTheme, }}>{children}</ThemeContext.Provider>;
}
