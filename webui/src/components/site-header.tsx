import { Link, useLocation, } from "react-router";
import { Activity, LayoutGrid, Settings, Sun, Moon, } from "lucide-react";
import { useContext, } from "react";
import { useTranslation, } from "react-i18next";
import { ThemeContext, } from "@/components/theme-context";
import { LanguageSwitcher, } from "@/components/language-switcher";
import { cn, } from "@/lib/utils";
import { Button, } from "@/components/ui/button";

const NAV = [
  { href: "/browse", labelKey: "nav.browse", icon: LayoutGrid, },
  { href: "/dashboard", labelKey: "nav.dashboard", icon: Settings, },
];

export function SiteHeader() {
  const pathname = useLocation().pathname;
  const { theme, setTheme, } = useContext(ThemeContext,);
  const { t, } = useTranslation();
  const isDark = theme === "dark";

  return (
    <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-screen-2xl items-center gap-6 px-4">
        <Link to="/" className="flex items-center gap-2 shrink-0">
          <Activity className="size-5 text-green" />
          <span className="font-mono text-base font-bold tracking-widest text-green">
            {t("nav.dhtCrawlerPrefix",)}<span className="text-foreground">::</span>{t("nav.dhtCrawlerSuffix",)}
          </span>
        </Link>

        <nav className="flex items-center gap-1">
          {NAV.map(({ href, labelKey, icon: Icon, },) => {
            const active = pathname === href || pathname.startsWith(href + "/",);
            return (
              <Link
                key={href}
                to={href}
                className={cn(
                  "flex items-center gap-1.5 rounded px-3 py-1.5 text-sm font-mono font-medium transition-colors",
                  active ? "bg-green/10 text-green" : "text-muted-foreground hover:text-foreground hover:bg-muted",
                )}
              >
                <Icon className="size-4" />
                {t(labelKey,)}
              </Link>
            );
          },)}
        </nav>

        <div className="ml-auto flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-muted-foreground hover:text-foreground"
            onClick={() => setTheme(isDark ? "light" : "dark",)}
            aria-label={t(isDark ? "nav.themeLight" : "nav.themeDark",)}
          >
            {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>
          <LanguageSwitcher />
        </div>
      </div>
    </header>
  );
}
