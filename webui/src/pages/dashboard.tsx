import { useNavigate, } from "react-router";
import { useTranslation, } from "react-i18next";
import { Tabs, TabsContent, TabsList, TabsTrigger, } from "@/components/ui/tabs";
import { CrawlerOverview, } from "@/components/dashboard/crawler-overview";
import { StatsPanel, } from "@/components/dashboard/stats-panel";
import { ConfigTabs, } from "@/components/dashboard/config-tabs";
import { Activity, BarChart2, Settings, } from "lucide-react";
import { useDocumentTitle, } from "@/lib/use-document-title";

interface DashboardPageProps {
  tab: string;
}

export function DashboardPage({ tab, }: DashboardPageProps,) {
  const { t, } = useTranslation();
  const navigate = useNavigate();

  const TAB_KEY: Record<string, string> = {
    overview: "dashboard.overview",
    stats: "dashboard.statistics",
    config: "dashboard.config",
  };

  useDocumentTitle(t("title.dashboard", { tab: t(TAB_KEY[tab],), },),);

  const TAB_TITLE_KEY: Record<string, string> = {
    overview: "dashboard.overviewTitle",
    stats: "dashboard.statisticsTitle",
    config: "dashboard.configTitle",
  };

  return (
    <main className="mx-auto max-w-screen-xl px-4 pt-6 pb-20 terminal-scanline">
      <div className="mb-6">
        <h1 className="font-mono text-2xl font-bold text-foreground">
          <span className="text-green">$</span> {t("dashboard.title",)}
          <span className="text-muted-foreground">::</span>
          {t(TAB_TITLE_KEY[tab] ?? tab,)}
        </h1>
        <p className="font-mono text-xs text-muted-foreground mt-1">{t("dashboard.subtitle",)}</p>
      </div>

      <Tabs value={tab} onValueChange={(v,) => navigate(`/dashboard/${v}`,)} className="w-full">
        <TabsList className="mb-4 font-mono bg-card border border-border h-9 gap-0.5 p-1">
          <TabsTrigger value="overview" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
            <Activity className="size-3.5" />
            {t("dashboard.overview",)}
          </TabsTrigger>
          <TabsTrigger value="stats" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
            <BarChart2 className="size-3.5" />
            {t("dashboard.statistics",)}
          </TabsTrigger>
          <TabsTrigger value="config" className="gap-1.5 text-xs data-active:bg-green/10 data-active:text-green">
            <Settings className="size-3.5" />
            {t("dashboard.config",)}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <CrawlerOverview />
        </TabsContent>
        <TabsContent value="stats">
          <StatsPanel />
        </TabsContent>
        <TabsContent value="config">
          <ConfigTabs />
        </TabsContent>
      </Tabs>
    </main>
  );
}
