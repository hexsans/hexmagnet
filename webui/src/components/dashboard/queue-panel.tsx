import { useState, useEffect, useMemo, } from "react";
import { useQuery, } from "urql";
import { useTranslation, } from "react-i18next";
import { ChevronLeft, ChevronRight, } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectTrigger, } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, } from "@/components/ui/table";
import { Badge, } from "@/components/ui/badge";
import { Button, } from "@/components/ui/button";
import { Skeleton, } from "@/components/ui/skeleton";
import { PageInput, } from "@/components/page-input";

import { QueueJobsDocument, type QueueJobsQueryInput, } from "@/lib/graphql/generated/graphql";
import { LIMIT_OPTIONS, } from "@/lib/constants";
import { getSP, } from "@/lib/url";

export function QueuePanel() {
  const { t, } = useTranslation();
  const [page, setPage,] = useState(() => {
    const p = parseInt(getSP().get("page",) || "1", 10,);
    return !isNaN(p,) && p > 0 ? p : 1;
  },);
  const [limit, setLimit,] = useState(() => {
    const l = parseInt(getSP().get("limit",) || "15", 10,);
    return LIMIT_OPTIONS.includes(l,) ? l : 15;
  },);
  useEffect(() => {
    const sp = new URLSearchParams(window.location.search,);
    if (page > 1) sp.set("page", String(page,),);
    else sp.delete("page",);
    if (limit !== 15) sp.set("limit", String(limit,),);
    else sp.delete("limit",);
    const str = sp.toString();
    window.history.replaceState(null, "", str ? `?${str}` : window.location.pathname,);
  }, [page, limit,],);

  const input: QueueJobsQueryInput = {
    limit,
    offset: (page - 1) * limit,
  };

  const [result, reexecute,] = useQuery({
    query: QueueJobsDocument,
    variables: { input, },
  },);

  useEffect(() => {
    const interval = setInterval(() => reexecute({ requestPolicy: "network-only", },), 5000,);
    return () => clearInterval(interval,);
  }, [reexecute,],);

  const { data, fetching, error, } = result;

  const jobs = data?.queue?.jobs;
  const items = useMemo(() => jobs?.items ?? [], [jobs?.items,],);
  const totalCount = jobs?.totalCount ?? 0;
  const hasNextPage = jobs?.hasNextPage ?? false;
  const totalPages = Math.max(1, Math.ceil(totalCount / limit,),);

  const isInitialLoad = fetching && items.length === 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="border-b border-border hover:bg-transparent">
              {[t("dashboard.processingQueue",), t("dashboard.queueMessages",),].map((h,) => (
                <TableHead key={h} className="font-mono text-xs text-muted-foreground uppercase tracking-wider bg-card/50">
                  {h}
                </TableHead>
              ),)}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isInitialLoad &&
              Array.from({ length: 8, },).map((_, i,) => (
                <TableRow key={i} className="border-border">
                  {Array.from({ length: 2, },).map((_, j,) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-full" />
                    </TableCell>
                  ),)}
                </TableRow>
              ),)}

            {!isInitialLoad && items.length === 0 && !error && (
              <TableRow>
                <TableCell colSpan={2} className="text-center font-mono text-sm text-muted-foreground py-12">
                  {t("dashboard.noJobs",)}
                </TableCell>
              </TableRow>
            )}

            {!fetching && error && (
              <TableRow>
                <TableCell colSpan={2} className="text-center font-mono text-sm text-destructive py-12">
                  {error.message}
                </TableCell>
              </TableRow>
            )}

            {items.map((job: { id: string; queue: string; payload: string },) => {
              let messagesCount: number | null = null;
              try {
                const parsed = JSON.parse(job.payload,);
                if (typeof parsed.lag === "number") {
                  messagesCount = parsed.lag;
                }
              } catch {
                /* empty */
              }
              return (
                <TableRow key={job.id} className="border-b border-border/50 transition-colors hover:bg-card/50">
                  <TableCell className="max-w-[180px]">
                    <span className="font-mono text-xs text-foreground truncate block" title={job.queue}>
                      {job.queue}
                    </span>
                  </TableCell>
                  <TableCell>
                    {messagesCount !== null ? (
                      <Badge
                        variant={messagesCount > 0 ? "secondary" : "outline"}
                        className={`font-mono text-xs py-0 h-4 ${messagesCount > 0 ? "" : "text-muted-foreground/40"}`}
                      >
                        {messagesCount.toLocaleString()}
                      </Badge>
                    ) : (
                      <span className="font-mono text-xs text-muted-foreground/40">{"\u2014"}</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            },)}
          </TableBody>
        </Table>
      </div>

      {totalPages > 1 && (
        <div className="flex flex-col items-center gap-3 sm:flex-row sm:justify-between">
          <div className="hidden flex-1 sm:block" />
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              className="font-mono text-xs border-border"
              disabled={page <= 1}
              onClick={() => setPage((p,) => Math.max(1, p - 1,),)}
            >
              <ChevronLeft className="size-3.5" />
              {t("common.prev",)}
            </Button>
            <span className="font-mono text-xs text-muted-foreground">
              <PageInput page={page} totalPages={totalPages} onPageChange={setPage} />
            </span>
            <Button
              variant="outline"
              size="sm"
              className="font-mono text-xs border-border"
              disabled={!hasNextPage}
              onClick={() => setPage((p,) => p + 1,)}
            >
              {t("common.next",)}
              <ChevronRight className="size-3.5" />
            </Button>
          </div>
          <div className="flex items-center justify-end gap-2 sm:flex-1">
            <span className="font-mono text-xs text-muted-foreground">{t("common.perPage",)}</span>
            <Select
              value={String(limit,)}
              onValueChange={(v,) => {
                setLimit(Number(v,),);
                setPage(1,);
              }}
            >
              <SelectTrigger className="w-20 font-mono text-xs border-border h-8">
                <span className="flex-1 text-left truncate">{limit}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-xs">
                {LIMIT_OPTIONS.map((n,) => (
                  <SelectItem key={n} value={String(n,)}>
                    {n}
                  </SelectItem>
                ),)}
              </SelectContent>
            </Select>
          </div>
        </div>
      )}
    </div>
  );
}
