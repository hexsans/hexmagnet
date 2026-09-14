import { useEffect, } from "react";
import { useTranslation, } from "react-i18next";
import { Zap, Users, Globe, } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { Badge, } from "@/components/ui/badge";
import { Skeleton, } from "@/components/ui/skeleton";
import { StatCard, } from "@/components/dashboard/stat-card";
import { ActivityFeed, } from "@/components/dashboard/activity-feed";
import { HealthCheckDocument, WorkersDocument, DhtCrawlerDocument, } from "@/lib/graphql/generated/graphql";
import { useQuery, } from "urql";
import { formatNumber, } from "@/lib/format";

export function CrawlerOverview() {
  const { t, } = useTranslation();

  const [healthResult, reexecuteHealth,] = useQuery({ query: HealthCheckDocument, },);
  const [workersResult, reexecuteWorkers,] = useQuery({ query: WorkersDocument, },);
  const [crawlerResult, reexecuteCrawler,] = useQuery({ query: DhtCrawlerDocument, },);

  useEffect(() => {
    const interval = setInterval(() => {
      reexecuteHealth({ requestPolicy: "network-only", },);
      reexecuteWorkers({ requestPolicy: "network-only", },);
      reexecuteCrawler({ requestPolicy: "network-only", },);
    }, 5000,);
    return () => clearInterval(interval,);
  }, [reexecuteHealth, reexecuteWorkers, reexecuteCrawler,],);

  const { data: healthData, fetching: healthFetching, } = healthResult;
  const { data: workersData, fetching: workersFetching, } = workersResult;
  const { data: crawlerData, fetching: crawlerFetching, } = crawlerResult;

  const health = healthData?.health;
  const version = healthData?.version;
  const workers = workersData?.workers?.listAll?.workers;
  const crawler = crawlerData?.dhtCrawler;

  const fetching = crawlerFetching;
  const status = crawler;
  const pauseReason =
    status?.pauseReason === "embedding reindex"
      ? t("dashboard.pauseReasonEmbeddingReindex",)
      : status?.pauseReason === "classifier reclassify"
        ? t("dashboard.pauseReasonReclassify",)
        : status?.pauseReason ?? "";

  const uptimeSeconds = Math.max(1, Number(status?.uptime ?? 1,),);
  const perMin = (val: number,) => Math.round(val / (uptimeSeconds / 60),);

  if (fetching && !status) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-14 w-full" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          {Array.from({ length: 4, },).map((_, i,) => (
            <Skeleton key={i} className="h-24 w-full" />
          ),)}
        </div>
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!status) return null;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-card px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          {healthFetching && !health ? (
            <Skeleton className="size-2.5 rounded-full" />
          ) : (
            <span className="relative flex size-2.5">
              <span
                className={
                  "absolute inline-flex size-full animate-ping rounded-full opacity-60 " +
                  (health?.status === "up" ? "bg-green" : health?.status === "down" ? "bg-red" : "bg-muted-foreground")
                }
              />
              <span
                className={
                  "relative inline-flex size-2.5 rounded-full " +
                  (health?.status === "up" ? "bg-green" : health?.status === "down" ? "bg-red" : "bg-muted-foreground")
                }
              />
            </span>
          )}
          <div>
            {healthFetching && !health ? (
              <Skeleton className="h-4 w-36" />
            ) : (
              <>
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <p className="font-mono text-sm font-bold text-foreground">
                    <span
                      className={health?.status === "up" ? "text-green" : health?.status === "down" ? "text-red" : "text-muted-foreground"}
                    >
                      {t("dashboard.systemStatus", { status: health?.status?.toUpperCase() ?? "UNKNOWN", },)}
                    </span>
                  </p>
                  {status.paused && (
                    <Badge variant="outline" className="h-auto min-h-4 max-w-full whitespace-normal border-amber/40 bg-amber/10 px-1.5 py-0.5 text-left font-mono text-[10px] leading-tight text-amber">
                      {t("dashboard.crawlerPaused", { reason: pauseReason, },)}
                    </Badge>
                  )}
                </div>
                {version && <p className="font-mono text-xs text-muted-foreground">{t("common.versionLabel", { version, },)}</p>}
              </>
            )}
          </div>
        </div>
        <div className="hidden sm:flex gap-2">
          {healthFetching && !health
            ? Array.from({ length: 3, },).map((_, i,) => <Skeleton key={i} className="h-6 w-24" />,)
            : health?.checks?.map((c,) => (
              <Badge key={c.key} variant={c.status === "up" ? "default" : "destructive"} className="font-mono text-xs">
                {c.key}: {c.status}
              </Badge>
            ),)}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
        <StatCard
          label={t("dashboard.torrentsCrawled",)}
          value={formatNumber(status.torrentsCrawled,)}
          sub={t("dashboard.perMin", { count: perMin(status.torrentsCrawled,), },)}
          icon={Zap}
          accent="green"
        />
        <StatCard
          label={t("dashboard.peersConnected",)}
          value={formatNumber(status.peersConnected,)}
          sub={t("dashboard.perMin", { count: perMin(status.peersConnected,), },)}
          icon={Users}
          accent="cyan"
        />
        <StatCard
          label={t("dashboard.peersDiscovered",)}
          value={formatNumber(status.peersDiscovered,)}
          sub={t("dashboard.perMin", { count: perMin(status.peersDiscovered,), },)}
          icon={Globe}
          accent="amber"
        />
      </div>

      <Card className="border-border">
        <CardHeader className="pb-2 pt-4 px-4">
          <CardTitle className="font-mono text-xs text-muted-foreground uppercase tracking-wider">{t("dashboard.workers",)}</CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          {workersFetching && !workers ? (
            <div className="flex flex-wrap gap-2">
              {Array.from({ length: 4, },).map((_, i,) => (
                <Skeleton key={i} className="h-7 w-28" />
              ),)}
            </div>
          ) : (
            <div className="flex flex-wrap gap-2">
              {workers?.length ? (
                workers.map((w,) => (
                  <Badge key={w.key} variant={w.started ? "default" : "secondary"} className="font-mono text-xs">
                    {w.key}
                  </Badge>
                ),)
              ) : (
                <span className="font-mono text-xs text-muted-foreground">{t("dashboard.noWorkers",)}</span>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="border-border">
        <CardHeader className="pb-2 pt-4 px-4">
          <CardTitle className="font-mono text-xs text-muted-foreground uppercase tracking-wider">
            {t("dashboard.recentActivity",)}
          </CardTitle>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          <ActivityFeed activity={status.recentActivity ?? []} />
        </CardContent>
      </Card>
    </div>
  );
}
