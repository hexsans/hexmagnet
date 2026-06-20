import { useQuery, } from "urql";
import { useTranslation, } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { TorrentMetricsDocument, } from "@/lib/graphql/generated/graphql";
import { QueuePanel, } from "@/components/dashboard/queue-panel";

export function StatsPanel() {
  const { t, } = useTranslation();
  const [metricsResult,] = useQuery({
    query: TorrentMetricsDocument,
    variables: {
      input: { bucketDuration: "day", },
    },
  },);

  const { data: metricsData, } = metricsResult;

  const buckets = metricsData?.torrent?.metrics?.buckets ?? [];

  const crawlHistory: { date: string; count: number; updatedCount: number }[] = buckets
    .map((b: { count: number; updatedCount: number; bucket: string },) => ({
      date: b.bucket ? new Date(b.bucket,).toLocaleDateString() : "",
      count: b.count,
      updatedCount: b.updatedCount,
    }),)
    .slice(0, 14,);

  return (
    <div className="flex flex-col gap-4">
      <Card className="border-border">
        <CardHeader className="pb-3 pt-4 px-4">
          <CardTitle className="font-mono text-xs text-muted-foreground uppercase tracking-wider">
            {t("dashboard.metricsHistory",)}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <div className="flex items-end gap-0.5 h-20">
            {crawlHistory.length === 0 && (
              <p className="font-mono text-xs text-muted-foreground w-full text-center py-8">{t("dashboard.noMetrics",)}</p>
            )}
            {crawlHistory.length > 0 &&
              crawlHistory.map(({ date, count, updatedCount, },) => {
                const max = Math.max(...crawlHistory.map((d,) => d.count,), 1,);
                const heightPct = Math.max((count / max) * 100, 4,);
                const hasUpdated = updatedCount > 0;
                return (
                  <div
                    key={date}
                    className="group relative flex-1 flex flex-col items-center justify-end h-full"
                  >
                    <div
                      className="w-full rounded-sm transition-all cursor-pointer"
                      style={{
                        height: `${heightPct}%`,
                        backgroundColor: "oklch(0.72 0.19 158 / 0.55)",
                      }}
                      onMouseEnter={(e,) => {
                        (e.currentTarget as HTMLDivElement).style.backgroundColor = "oklch(0.72 0.19 158 / 1)";
                      }}
                      onMouseLeave={(e,) => {
                        (e.currentTarget as HTMLDivElement).style.backgroundColor = "oklch(0.72 0.19 158 / 0.55)";
                      }}
                    />
                    <div className="absolute -top-7 left-1/2 -translate-x-1/2 hidden group-hover:block z-10 pointer-events-none">
                      <div className="rounded bg-card border border-border px-1.5 py-0.5 font-mono text-xs text-foreground whitespace-nowrap">
                        {date}: {count.toLocaleString()} total{hasUpdated && `, ${updatedCount.toLocaleString()} updated`}
                      </div>
                    </div>
                  </div>
                );
              },)}
          </div>
        </CardContent>
      </Card>

      <QueuePanel />
    </div>
  );
}
