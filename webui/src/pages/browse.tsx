import { useState, useCallback, useRef, useEffect, useMemo, useReducer, } from "react";
import { useQuery, } from "urql";
import { useTranslation, } from "react-i18next";
import { Search, Filter, RefreshCw, ChevronLeft, ChevronRight, ArrowUpDown, Plus, X, } from "lucide-react";
import { Input, } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, } from "@/components/ui/select";
import { Button, } from "@/components/ui/button";
import { Separator, } from "@/components/ui/separator";
import { TorrentList, } from "@/components/torrent-list";
import { PageInput, } from "@/components/page-input";
import { TorrentDetailDialog, } from "@/components/torrent-detail-dialog";
import { BulkActionsToolbar, } from "@/components/browse/bulk-actions-toolbar";
import {
  TorrentSearchDocument,
  type TorrentSearchQueryInput,
  type ContentType,
  type TorrentSearchOrderByField,
} from "@/lib/graphql/generated/graphql";
import { toTorrent, } from "@/lib/graphql/adapters";
import { useDocumentTitle, } from "@/lib/use-document-title";
import { cn, } from "@/lib/utils";
import { LIMIT_OPTIONS, } from "@/lib/constants";
import { getSP, } from "@/lib/url";
import type { Torrent, } from "@/lib/types";

const CATEGORY_KEYS: Record<string, string> = {
  movie: "category.movie",
  tv_show: "category.tvShow",
  music: "category.music",
  ebook: "category.ebook",
  comic: "category.comic",
  audiobook: "category.audiobook",
  game: "category.game",
  software: "category.software",
  adult: "category.adult",
  other: "category.other",
};

type SortRule = { field: string; direction: "asc" | "desc" };

const FIELD_OPTIONS: { value: string; i18nKey: string }[] = [
  { value: "created_at", i18nKey: "sort.createdAt", },
  { value: "size", i18nKey: "sort.size", },
  { value: "seeders", i18nKey: "sort.seeders", },
  { value: "leechers", i18nKey: "sort.leechers", },
  { value: "name", i18nKey: "sort.name", },
  { value: "relevance", i18nKey: "sort.relevance", },
];

const FIELD_LABELS: Record<string, string> = Object.fromEntries(FIELD_OPTIONS.map((o,) => [o.value, o.i18nKey,],),);

function defaultSortRules(): SortRule[] {
  return [{ field: "created_at", direction: "desc", },];
}

function parseSortRules(raw: string | null,): SortRule[] {
  if (!raw) return [];
  return raw
    .split(",",)
    .map((s,) => {
      const parts = s.split(":",);
      return { field: parts[0] ?? "created_at", direction: (parts[1] ?? "desc") as "asc" | "desc", };
    },)
    .filter((r,) => FIELD_LABELS[r.field],);
}

function formatSortRules(rules: SortRule[],): string {
  return rules.map((r,) => `${r.field}:${r.direction}`,).join(",",);
}

function sortRulesEqual(a: SortRule[], b: SortRule[],): boolean {
  if (a.length !== b.length) return false;
  return a.every((r, i,) => r.field === b[i].field && r.direction === b[i].direction,);
}

type SelectionAction =
  | { type: "toggle"; infoHash: string }
  | { type: "toggleAll"; hashes: string[] }
  | { type: "clear" }
  | { type: "sync"; currentHashes: Set<string> };

function selectionReducer(state: Set<string>, action: SelectionAction,): Set<string> {
  switch (action.type) {
    case "toggle": {
      const next = new Set(state,);
      if (next.has(action.infoHash,)) next.delete(action.infoHash,);
      else next.add(action.infoHash,);
      return next;
    }
    case "toggleAll": {
      if (state.size === action.hashes.length) return new Set();
      return new Set(action.hashes,);
    }
    case "clear":
      return new Set();
    case "sync": {
      const next = new Set([...state,].filter((h,) => action.currentHashes.has(h,),),);
      return next.size === state.size ? state : next;
    }
  }
}

export function BrowsePage() {
  const { t, } = useTranslation();
  useDocumentTitle(t("title.browse",),);
  const [query, setQuery,] = useState(() => getSP().get("q",) || "",);
  const [debouncedQuery, setDebouncedQuery,] = useState(() => (getSP().get("q",) || "").trim(),);
  const [categories, setCategories,] = useState<string[]>(() => {
    const raw = getSP().get("categories",);
    return raw ? raw.split(",",) : [];
  },);
  const [categoryOpen, setCategoryOpen,] = useState(false,);
  const [page, setPage,] = useState(() => {
    const p = parseInt(getSP().get("page",) || "1", 10,);
    return !isNaN(p,) && p > 0 ? p : 1;
  },);
  const [limit, setLimit,] = useState(() => {
    const l = parseInt(getSP().get("limit",) || "15", 10,);
    return LIMIT_OPTIONS.includes(l,) ? l : 15;
  },);
  const spSort = getSP().get("sort",);
  const [sortRules, setSortRules,] = useState<SortRule[]>(() => {
    return spSort ? parseSortRules(spSort,) : defaultSortRules();
  },);
  const [sortOpen, setSortOpen,] = useState(false,);
  const [detailTorrent, setDetailTorrent,] = useState<Torrent | null>(null,);
  const [detailOpen, setDetailOpen,] = useState(false,);
  const [selectedIds, dispatch,] = useReducer(selectionReducer, new Set<string>(),);

  const [barrier, setBarrier,] = useState<string | null>(() => {
    return getSP().get("barrier",) || null;
  },);

  const isDefaultSort = sortRulesEqual(sortRules, defaultSortRules(),);
  const hasActiveNarrowing = !!debouncedQuery || categories.length > 0 || !isDefaultSort;
  const shouldUseBarrier = hasActiveNarrowing || page > 1;

  // Clear barrier when query inputs change (search, categories, sort, limit)
  const queryKey = `${debouncedQuery}|${categories.join(",",)}|${formatSortRules(sortRules,)}|${limit}`;
  const prevQueryKey = useRef(queryKey,);
  useEffect(() => {
    if (prevQueryKey.current !== queryKey) {
      prevQueryKey.current = queryKey;
      setBarrier(null,);
    }
  }, [queryKey,],);

  const prevQueryRef = useRef(debouncedQuery,);
  useEffect(() => {
    if (prevQueryRef.current && !debouncedQuery) {
      setSortRules((prev,) => {
        if (prev.some((r,) => r.field === "relevance",)) {
          return defaultSortRules();
        }
        return prev;
      },);
    }
    prevQueryRef.current = debouncedQuery;
  }, [debouncedQuery,],);

  const toggleSelect = useCallback((infoHash: string,) => {
    dispatch({ type: "toggle", infoHash, },);
  }, [],);

  const clearSelection = useCallback(() => {
    dispatch({ type: "clear", },);
  }, [],);

  useEffect(() => {
    const sp = new URLSearchParams(window.location.search,);
    if (debouncedQuery) sp.set("q", debouncedQuery,);
    else sp.delete("q",);
    if (categories.length > 0) sp.set("categories", categories.join(",",),);
    else sp.delete("categories",);
    if (page > 1) sp.set("page", String(page,),);
    else sp.delete("page",);
    if (limit !== 15) sp.set("limit", String(limit,),);
    else sp.delete("limit",);
    if (barrier && shouldUseBarrier) sp.set("barrier", barrier,);
    else sp.delete("barrier",);
    const defaultRules = defaultSortRules();
    const formatted = formatSortRules(sortRules,);
    if (!sortRulesEqual(sortRules, defaultRules,) && formatted) sp.set("sort", formatted,);
    else sp.delete("sort",);
    const str = sp.toString();
    window.history.replaceState(null, "", str ? `?${str}` : window.location.pathname,);
  }, [debouncedQuery, categories, page, limit, sortRules, barrier, shouldUseBarrier,],);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null,);
  const debouncedQueryRef = useRef((getSP().get("q",) || "").trim(),);

  const handleSearch = useCallback((val: string,) => {
    setQuery(val,);
    if (debounceRef.current) clearTimeout(debounceRef.current,);
    debounceRef.current = setTimeout(() => {
      const trimmed = val.trim();
      if (trimmed !== debouncedQueryRef.current) {
        debouncedQueryRef.current = trimmed;
        setDebouncedQuery(trimmed,);
      }
      setPage(1,);
    }, 300,);
  }, [],);

  const input: TorrentSearchQueryInput = {
    limit,
    page,
    hasNextPage: true,
    totalCount: true,
  };
  if (barrier && shouldUseBarrier) {
    input.barrier = barrier;
  }
  const activeRules = sortRules;
  if (activeRules.length > 0) {
    input.orderBy = activeRules.map((r,) => ({
      field: r.field as TorrentSearchOrderByField,
      direction: r.direction,
    }),);
  }

  if (debouncedQuery) input.queryString = debouncedQuery;
  const facets: TorrentSearchQueryInput["facets"] = {
    contentType: { aggregate: true, },
  };
  if (categories.length > 0) {
    facets.contentType!.filter = categories as ContentType[];
  }
  input.facets = facets;

  const [result, reexecute,] = useQuery({
    query: TorrentSearchDocument,
    variables: { input, },
    requestPolicy: "cache-and-network",
  },);

  const { data, fetching, error, } = result;

  const serverBarrier = data?.torrentSearch?.search?.barrier;
  const prevServerBarrier = useRef<string | null | undefined>(undefined,);
  if (serverBarrier !== prevServerBarrier.current) {
    if (serverBarrier && shouldUseBarrier) {
      setBarrier(serverBarrier,);
    }
    prevServerBarrier.current = serverBarrier;
  }

  const items = useMemo(() => data?.torrentSearch?.search?.items ?? [], [data?.torrentSearch?.search?.items,],);
  const totalCount = data?.torrentSearch?.search?.totalCount ?? 0;
  const hasNextPage = data?.torrentSearch?.search?.hasNextPage ?? false;

  const aggMap = useMemo(() => {
    const aggs = data?.torrentSearch?.search?.aggregations?.contentType ?? [];
    return new Map(aggs.map((a,) => [a.value as string, a.count,],),);
  }, [data?.torrentSearch?.search?.aggregations?.contentType,],);

  const categoryOptions = useMemo(() => {
    return Object.entries(CATEGORY_KEYS,).map(([value, i18nKey,],) => ({
      value,
      i18nKey,
      count: aggMap.get(value,) ?? 0,
    }),);
  }, [aggMap,],);

  const torrents = items.map(toTorrent,);
  const totalPages = Math.max(1, Math.ceil(totalCount / limit,),);

  const isInitialLoad = fetching && items.length === 0;
  const hasError = !!error;

  const toggleSelectAll = useCallback(() => {
    dispatch({
      type: "toggleAll",
      hashes: torrents.map((t,) => t.infoHash,),
    },);
  }, [torrents,],);

  useEffect(() => {
    dispatch({
      type: "sync",
      currentHashes: new Set(torrents.map((t,) => t.infoHash,),),
    },);
  }, [torrents,],);

  const selectedTorrents = useMemo(() => torrents.filter((t,) => selectedIds.has(t.infoHash,),), [torrents, selectedIds,],);
  const refresh = useCallback(() => {
    reexecute({ requestPolicy: "network-only", },);
  }, [reexecute,],);

  return (
    <main className="mx-auto max-w-screen-xl px-4 pt-6 pb-20">
      <div className="mb-6 flex items-end justify-between">
        <div>
          <h1 className="font-mono text-2xl font-bold text-foreground">
            <span className="text-green">$</span> {t("browse.title",)}
          </h1>
          <p className="font-mono text-xs text-muted-foreground mt-1">
            <span className={fetching ? "text-muted-foreground" : "text-green"}>{totalCount.toLocaleString()}</span> {t("browse.results",)}
            {debouncedQuery && (
              <>
                {" "}
                &mdash; {t("browse.queryLabel",)} <span className="text-cyan">&quot;{debouncedQuery}&quot;</span>
              </>
            )}
          </p>
        </div>
      </div>

      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e,) => handleSearch(e.target.value,)}
            placeholder={t("browse.searchPlaceholder",)}
            className="pl-9 pr-9 font-mono text-sm bg-card border-border focus-visible:ring-green/50 focus-visible:border-green/50"
          />
          {query.length > 0 && (
            <button
              onClick={() => handleSearch("",)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              aria-label={t("browse.clearSearch",)}
            >
              <X className="size-4" />
            </button>
          )}
        </div>

        <div className="relative">
          <Button
            variant="outline"
            className={cn(
              "shrink-0 gap-1.5 font-mono text-sm font-normal border-border bg-card",
              categoryOpen && "ring-1 ring-green/50 border-green/50",
            )}
            onClick={() => setCategoryOpen((v,) => !v,)}
          >
            <Filter className="size-3.5 text-muted-foreground shrink-0" />
            {categories.length === 0 || categories.length === categoryOptions.length
              ? t("browse.allCategories",)
              : categories.length === 1
                ? t("browse.category_one", { count: 1, },)
                : t("browse.category_other", { count: categories.length, },)}
          </Button>
          {categoryOpen && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setCategoryOpen(false,)} />
              <div className="absolute right-0 top-full mt-2 z-50 w-56 rounded-lg border border-border bg-card p-2 shadow-lg">
                {categoryOptions.map((cat,) => {
                  const selected = categories.includes(cat.value,);
                  return (
                    <button
                      key={cat.value}
                      className={cn(
                        "flex w-full items-center gap-2 rounded px-2 py-1.5 text-left font-mono text-sm cursor-pointer transition-colors",
                        selected ? "text-foreground" : "text-muted-foreground hover:text-foreground",
                      )}
                      onClick={() => {
                        setCategories((prev,) => (selected ? prev.filter((c,) => c !== cat.value,) : [...prev, cat.value,]),);
                        setPage(1,);
                      }}
                    >
                      <span
                        className={cn(
                          "flex size-4 shrink-0 items-center justify-center rounded border transition-colors",
                          selected ? "border-green bg-green text-white" : "border-border",
                        )}
                      >
                        {selected && (
                          <svg className="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
                            <path d="M20 6L9 17l-5-5" />
                          </svg>
                        )}
                      </span>
                      <span className="flex-1">{t(cat.i18nKey,)}</span>
                      <span className="text-xs text-muted-foreground/60">{cat.count.toLocaleString()}</span>
                    </button>
                  );
                },)}
                {categories.length > 0 && (
                  <>
                    <Separator className="my-1" />
                    <button
                      className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left font-mono text-xs text-muted-foreground hover:text-destructive cursor-pointer transition-colors"
                      onClick={() => {
                        setCategories([],);
                        setPage(1,);
                      }}
                    >
                      <X className="size-3" />
                      {t("browse.clearFilters",)}
                    </button>
                  </>
                )}
              </div>
            </>
          )}
        </div>

        <div className="relative">
          <Button
            variant="outline"
            className={cn(
              "gap-1.5 font-mono text-sm font-normal border-border bg-card",
              sortOpen && "ring-1 ring-green/50 border-green/50",
            )}
            onClick={() => setSortOpen((v,) => !v,)}
          >
            <ArrowUpDown className="size-3.5 text-muted-foreground" />
            {t("browse.sortLabel",)}
            {sortRules.length > 0 && (
              <span className="text-xs text-muted-foreground">({t("browse.sort", { count: sortRules.length, },)})</span>
            )}
          </Button>
          {sortOpen && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setSortOpen(false,)} />
              <div className="absolute right-0 top-full mt-2 z-50 w-72 rounded-lg border border-border bg-card p-3 shadow-lg">
                <p className="mb-2 font-mono text-sm font-semibold text-muted-foreground">{t("browse.sortBy",)}</p>
                <div className="space-y-2">
                  {sortRules.map((rule, i,) => {
                    const otherFields = new Set(sortRules.filter((_, j,) => j !== i,).map((r,) => r.field,),);
                    const availableOptions = FIELD_OPTIONS.filter((o,) => !otherFields.has(o.value,) && (debouncedQuery || o.value !== "relevance"),);
                    return (
                      <div key={i} className="flex items-center gap-2">
                        <Select
                          value={rule.field}
                          onValueChange={(v,) => {
                            setSortRules((prev,) => {
                              const next = [...prev,];
                              next[i] = { ...next[i], field: v ?? next[i].field, };
                              return next;
                            },);
                            setPage(1,);
                          }}
                        >
                          <SelectTrigger className="h-8 flex-1 font-mono text-sm border-border">
                            <span className="truncate">{t(FIELD_LABELS[rule.field],)}</span>
                          </SelectTrigger>
                          <SelectContent className="font-mono text-sm">
                            {availableOptions.map((o,) => (
                              <SelectItem key={o.value} value={o.value}>
                                {t(o.i18nKey,)}
                              </SelectItem>
                            ),)}
                          </SelectContent>
                        </Select>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-7 shrink-0 text-muted-foreground hover:text-green"
                          onClick={() => {
                            setSortRules((prev,) => {
                              const next = [...prev,];
                              next[i] = { ...next[i], direction: next[i].direction === "desc" ? "asc" : "desc", };
                              return next;
                            },);
                            setPage(1,);
                          }}
                        >
                          {rule.direction === "desc" ? <span className="text-xs font-bold">▼</span> : <span className="text-xs font-bold">▲</span>}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className={cn(
                            "size-7 shrink-0",
                            sortRules.length > 1
                              ? "text-muted-foreground hover:text-destructive"
                              : "text-muted-foreground/20 cursor-not-allowed",
                          )}
                          disabled={sortRules.length <= 1}
                          onClick={() => {
                            if (sortRules.length <= 1) return;
                            setSortRules((prev,) => prev.filter((_, j,) => j !== i,),);
                            setPage(1,);
                          }}
                        >
                          <X className="size-3" />
                        </Button>
                      </div>
                    );
                  },)}
                </div>
                {(() => {
                  const usedFields = new Set(sortRules.map((r,) => r.field,),);
                  const unused = FIELD_OPTIONS.filter((o,) => !usedFields.has(o.value,) && (debouncedQuery || o.value !== "relevance"),);
                  return (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="mt-2 w-full gap-1 font-mono text-sm text-muted-foreground hover:text-green"
                      disabled={unused.length === 0}
                      onClick={() => {
                        setSortRules((prev,) => [...prev, { field: unused[0].value, direction: "desc", },],);
                        setPage(1,);
                      }}
                    >
                      <Plus className="size-3" />
                      {t("browse.addSort",)}
                    </Button>
                  );
                })()}
              </div>
            </>
          )}
        </div>
      </div>

      <div
        className={`overflow-hidden transition-all duration-300 ease-in-out ${
          selectedTorrents.length > 0
            ? "max-h-20 opacity-100 mb-3"
            : "max-h-0 opacity-0 mb-0"
        }`}
        aria-hidden={selectedTorrents.length === 0}
      >
        <BulkActionsToolbar selectedTorrents={selectedTorrents} onMutate={refresh} onClearSelection={clearSelection} />
      </div>

      {hasError && items.length === 0 ? (
        <div className="rounded-lg border border-destructive/50 bg-destructive/5 p-8 text-center">
          <p className="font-mono text-lg font-bold text-destructive mb-2">{t("browse.searchError",)}</p>
          <p className="font-mono text-sm text-muted-foreground mb-4">{t("browse.searchErrorMessage",)}</p>
          <Button
            variant="outline"
            size="sm"
            className="font-mono text-sm border-destructive/50 text-destructive hover:bg-destructive/10"
            onClick={refresh}
          >
            <RefreshCw className="size-3 mr-1.5" />
            {t("common.retry",)}
          </Button>
        </div>
      ) : (
        <>
          {hasError && items.length > 0 && (
            <div className="mb-3 flex items-center gap-2 rounded-lg border border-amber/50 bg-amber/5 px-4 py-2.5">
              <p className="flex-1 font-mono text-sm text-amber">{t("browse.showingCached",)}</p>
              <Button
                variant="ghost"
                size="sm"
                className="font-mono text-xs text-amber hover:text-amber/80 hover:bg-amber/10 shrink-0"
                onClick={refresh}
              >
                <RefreshCw className="size-3 mr-1" />
                {t("common.retry",)}
              </Button>
            </div>
          )}

          <TorrentList
            torrents={torrents}
            isLoading={isInitialLoad}
            onMutate={refresh}
            onClickDetail={(t,) => {
              setDetailTorrent(t,);
              setDetailOpen(true,);
            }}
            selectedIds={selectedIds}
            onToggleSelect={toggleSelect}
            onToggleSelectAll={toggleSelectAll}
          />
        </>
      )}

      {totalPages > 1 && (
        <div className="mt-6 flex items-center justify-between gap-3 pb-6">
          <div className="flex-1" />
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              className="font-mono text-sm border-border"
              disabled={page <= 1}
              onClick={() => setPage((p,) => Math.max(1, p - 1,),)}
            >
              <ChevronLeft className="size-3.5" />
              {t("common.prev",)}
            </Button>
            <span className="font-mono text-sm text-muted-foreground">
              <PageInput page={page} totalPages={totalPages} onPageChange={setPage} />
            </span>
            <Button
              variant="outline"
              size="sm"
              className="font-mono text-sm border-border"
              disabled={!hasNextPage}
              onClick={() => setPage((p,) => p + 1,)}
            >
              {t("common.next",)}
              <ChevronRight className="size-3.5" />
            </Button>
          </div>
          <div className="flex flex-1 items-center justify-end gap-2">
            <span className="font-mono text-sm text-muted-foreground">{t("browse.perPage",)}</span>
            <Select
              value={String(limit,)}
              onValueChange={(v,) => {
                setLimit(Number(v,),);
                setPage(1,);
              }}
            >
              <SelectTrigger className="w-20 font-mono text-sm border-border h-8">
                <span className="flex-1 text-left truncate">{limit}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
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

      <TorrentDetailDialog torrent={detailTorrent} open={detailOpen} onOpenChange={setDetailOpen} />
    </main>
  );
}
