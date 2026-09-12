import { useState, useRef, useEffect, } from "react";
import { useQuery, useMutation, } from "urql";
import { useTranslation, } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { Input, } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, } from "@/components/ui/select";
import { Checkbox, } from "@/components/ui/checkbox";
import { Skeleton, } from "@/components/ui/skeleton";
import { LoaderCircle, RefreshCw, } from "lucide-react";
import { Button, } from "@/components/ui/button";
import { toast, } from "sonner";
import { ConfigDocument, UpdateConfigDocument, ReclassifyTorrentsDocument, } from "@/lib/graphql/generated/graphql";
import type { ConfigQuery, ClassifierConfigInput, } from "@/lib/graphql/generated/graphql";
import { useMaintenanceStatus, } from "@/lib/use-maintenance-status";

type ClassifierSection = ConfigQuery["config"]["classifier"];

export function ClassifierLLMConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<ClassifierSection | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.classifier ?? null;
  const active = form ?? original;

  const [, reclassifyMutation,] = useMutation(ReclassifyTorrentsDocument,);
  const { reclassify, reindexRunning, refresh: refreshMaintenance, } = useMaintenanceStatus();

  const reclassifyRunning = Boolean(reclassify?.running && !reclassify?.done,);
  const reclassifyTotal = reclassify?.total ?? 0;
  const reclassifyProcessed = reclassify?.processed ?? 0;
  const reclassifyBusy = reclassifyRunning || reindexRunning;
  const reclassifyErrorRef = useRef<string | null>(null,);
  const reclassifySawRunningRef = useRef(false,);
  const reclassifyCompletedRef = useRef(false,);

  function startReclassify() {
    reclassifyMutation({},).then((result,) => {
      const p = result.data?.torrent?.reclassifyTorrents;
      if (!p) return;
      if (p.error) { toast.error(p.error,); return; }
      if (p.done && p.processed > 0) { toast.success(t("dashboard.reclassifyDone", { count: p.processed, },),); }
      refreshMaintenance();
    },);
  }

  useEffect(() => {
    if (!reclassify) return;

    if (reclassify.error) {
      if (reclassify.error !== reclassifyErrorRef.current) {
        reclassifyErrorRef.current = reclassify.error;
        toast.error(reclassify.error,);
      }
      return;
    }

    reclassifyErrorRef.current = null;

    if (reclassify.running && !reclassify.done) {
      reclassifySawRunningRef.current = true;
      reclassifyCompletedRef.current = false;
      return;
    }

    if (reclassifySawRunningRef.current && reclassify.done && reclassify.processed > 0 && !reclassifyCompletedRef.current) {
      reclassifyCompletedRef.current = true;
      toast.success(t("dashboard.reclassifyDone", { count: reclassify.processed, },),);
    }
  }, [reclassify, t,],);

  if (fetching || !active) {
    return (
      <Card className="border-border">
        <CardHeader>
          <Skeleton className="h-4 w-40" />
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            {Array.from({ length: 4, },).map((_, i,) => (
              <Skeleton key={i} className="h-10 w-full" />
            ),)}
          </div>
        </CardContent>
      </Card>
    );
  }

  function updateLLM(partial: Partial<ClassifierSection["llm"]>,) {
    setForm({ ...active!, llm: { ...active!.llm, ...partial, }, } as ClassifierSection,);
  }

  function updateTorrentFilter(partial: Partial<ClassifierSection["torrentFilter"]>,) {
    setForm({ ...active!, torrentFilter: { ...active!.torrentFilter, ...partial, }, } as ClassifierSection,);
  }

  function updateTmdb(partial: Partial<ClassifierSection["tmdb"]>,) {
    setForm({ ...active!, tmdb: { ...active!.tmdb, ...partial, }, } as ClassifierSection,);
  }

  async function save() {
    if (!form || saving || !original) return;
    setSaving(true,);
    try {
      const llmPatch: Record<string, unknown> = {};
      if (form.llm.endpoint !== original.llm.endpoint) llmPatch.endpoint = form.llm.endpoint;
      if (form.llm.apiKey !== original.llm.apiKey) llmPatch.apiKey = form.llm.apiKey;
      if (form.llm.model !== original.llm.model) llmPatch.model = form.llm.model;
      if (form.llm.timeout !== original.llm.timeout) llmPatch.timeout = form.llm.timeout;
      if (form.llm.maxRetries !== original.llm.maxRetries) llmPatch.maxRetries = form.llm.maxRetries;
      if (form.llm.temperature !== original.llm.temperature) llmPatch.temperature = form.llm.temperature;
      if (form.llm.reasoningEffort !== original.llm.reasoningEffort) llmPatch.reasoningEffort = form.llm.reasoningEffort;
      if (form.llm.maxFiles !== original.llm.maxFiles) llmPatch.maxFiles = form.llm.maxFiles;
      if (form.llm.enabled !== original.llm.enabled) llmPatch.enabled = form.llm.enabled;

      const tfPatch: Record<string, unknown> = {};
      if (form.torrentFilter.mode !== original.torrentFilter.mode) tfPatch.mode = form.torrentFilter.mode;
      if (JSON.stringify(form.torrentFilter.titlePatterns,) !== JSON.stringify(original.torrentFilter.titlePatterns,)) {
        tfPatch.titlePatterns = form.torrentFilter.titlePatterns.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (JSON.stringify(form.torrentFilter.filenamePatterns,) !== JSON.stringify(original.torrentFilter.filenamePatterns,)) {
        tfPatch.filenamePatterns = form.torrentFilter.filenamePatterns.map((s,) => s.trim(),).filter(Boolean,);
      }

      const classifierPatch: Record<string, unknown> = {};
      if (form.concurrency !== original.concurrency) classifierPatch.concurrency = form.concurrency;
      if (Object.keys(llmPatch,).length > 0) classifierPatch.llm = llmPatch;
      if (Object.keys(tfPatch,).length > 0) classifierPatch.torrentFilter = tfPatch;

      const tmdbPatch: Record<string, unknown> = {};
      if (form.tmdb.accessToken !== original.tmdb.accessToken) tmdbPatch.accessToken = form.tmdb.accessToken;
      if (form.tmdb.enabled !== original.tmdb.enabled) tmdbPatch.enabled = form.tmdb.enabled;
      if (form.tmdb.rateLimit !== original.tmdb.rateLimit) tmdbPatch.rateLimit = form.tmdb.rateLimit;
      if (Object.keys(tmdbPatch,).length > 0) classifierPatch.tmdb = tmdbPatch;

      const result = await updateConfig({ input: { classifier: classifierPatch as ClassifierConfigInput, }, },);
      if (result.error) {
        toast.error(result.error.message,);
        return;
      }
      setForm(null,);
      reexecQuery({ requestPolicy: "network-only", },);
      toast.success(t("dashboard.saved",),);
    } catch {
      toast.error(t("dashboard.saveFailed",),);
    } finally {
      setSaving(false,);
    }
  }

  const effortLabels: Record<string, string> = {
    "": t("dashboard.llmReasoningEffortDefault",),
    none: t("dashboard.llmReasoningEffortNone",),
    low: t("dashboard.llmReasoningEffortLow",),
    medium: t("dashboard.llmReasoningEffortMedium",),
    high: t("dashboard.llmReasoningEffortHigh",),
  };

  return (
    <Card className="border-border">
      <CardHeader className="pb-0">
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.classifierConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.classifier",)}</legend>

          <div>
            <label htmlFor="classifierConcurrency" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.concurrency",)}
            </label>
            <Input
              id="classifierConcurrency"
              type="number"
              min={1}
              value={active.concurrency}
              onChange={(e,) => {
                const next = { ...active!, concurrency: parseInt(e.target.value, 10,) || 0, } as ClassifierSection;
                setForm(next,);
              }}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.concurrencyHint",)}</span>
          </div>

          <div className="flex flex-col">
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={active.llm.enabled}
                onCheckedChange={(v: boolean,) => updateLLM({ enabled: v, },)}
                className="size-4"
              />
              <span className="font-mono text-xs text-foreground uppercase">{t("dashboard.llmEnabled",)}</span>
            </label>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmEnabledHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.llm.enabled
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col gap-4 px-0.5">
                <div className="mt-4">
                  <label htmlFor="llmEndpoint" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmEndpoint",)}
                  </label>
                  <Input
                    id="llmEndpoint"
                    type="text"
                    value={active.llm.endpoint}
                    onChange={(e,) => updateLLM({ endpoint: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmEndpointHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmApiKey" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmApiKey",)}
                  </label>
                  <Input
                    id="llmApiKey"
                    type="text"
                    value={active.llm.apiKey}
                    onChange={(e,) => updateLLM({ apiKey: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmApiKeyHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmModel" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmModel",)}
                  </label>
                  <Input
                    id="llmModel"
                    type="text"
                    value={active.llm.model}
                    onChange={(e,) => updateLLM({ model: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmModelHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmTimeout" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmTimeout",)}
                  </label>
                  <Input
                    id="llmTimeout"
                    type="number"
                    value={active.llm.timeout}
                    onChange={(e,) => updateLLM({ timeout: parseInt(e.target.value, 10,) || 0, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmTimeoutHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmMaxRetries" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmMaxRetries",)}
                  </label>
                  <Input
                    id="llmMaxRetries"
                    type="number"
                    value={active.llm.maxRetries}
                    onChange={(e,) => updateLLM({ maxRetries: parseInt(e.target.value, 10,) || 0, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmMaxRetriesHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmTemperature" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmTemperature",)}
                  </label>
                  <Input
                    id="llmTemperature"
                    type="number"
                    step={0.1}
                    min={0}
                    max={2}
                    value={active.llm.temperature}
                    onChange={(e,) => updateLLM({ temperature: parseFloat(e.target.value,) || 0, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmTemperatureHint",)}</span>
                </div>

                <div>
                  <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.llmReasoningEffort",)}</label>
                  <Select value={active.llm.reasoningEffort} onValueChange={(v,) => v != null && updateLLM({ reasoningEffort: v, },)}>
                    <SelectTrigger className="w-full font-mono text-sm border-border">
                      <span className="flex-1 text-left truncate">{effortLabels[active.llm.reasoningEffort]}</span>
                    </SelectTrigger>
                    <SelectContent className="font-mono text-sm">
                      <SelectItem value="">{t("dashboard.llmReasoningEffortDefault",)}</SelectItem>
                      <SelectItem value="none">{t("dashboard.llmReasoningEffortNone",)}</SelectItem>
                      <SelectItem value="low">{t("dashboard.llmReasoningEffortLow",)}</SelectItem>
                      <SelectItem value="medium">{t("dashboard.llmReasoningEffortMedium",)}</SelectItem>
                      <SelectItem value="high">{t("dashboard.llmReasoningEffortHigh",)}</SelectItem>
                    </SelectContent>
                  </Select>
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmReasoningEffortHint",)}</span>
                </div>

                <div>
                  <label htmlFor="llmMaxFiles" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.llmMaxFiles",)}
                  </label>
                  <Input
                    id="llmMaxFiles"
                    type="number"
                    min={-1}
                    value={active.llm.maxFiles}
                    onChange={(e,) => updateLLM({ maxFiles: parseInt(e.target.value, 10,) || 0, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.llmMaxFilesHint",)}</span>
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.torrentFilter",)}</legend>

          <div>
            <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.torrentFilterMode",)}</label>
            <Select value={active.torrentFilter.mode} onValueChange={(v,) => v != null && updateTorrentFilter({ mode: v, },)}>
              <SelectTrigger className="w-full font-mono text-sm border-border">
                <span className="flex-1 text-left truncate">
                  {active.torrentFilter.mode === "off" && t("dashboard.torrentFilterModeOff",)}
                  {active.torrentFilter.mode === "process" && t("dashboard.torrentFilterModeProcess",)}
                  {active.torrentFilter.mode === "discard" && t("dashboard.torrentFilterModeDiscard",)}
                </span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
                <SelectItem value="off">{t("dashboard.torrentFilterModeOff",)}</SelectItem>
                <SelectItem value="process">{t("dashboard.torrentFilterModeProcess",)}</SelectItem>
                <SelectItem value="discard">{t("dashboard.torrentFilterModeDiscard",)}</SelectItem>
              </SelectContent>
            </Select>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torrentFilterModeHint",)}</span>
          </div>

          <div className="flex flex-col">
            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.torrentFilter.mode !== "off"
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col gap-4 px-0.5">
                <div className="mt-4">
                  <label htmlFor="torrentFilterTitlePatterns" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.torrentFilterTitlePatterns",)}</label>
                  <textarea
                    id="torrentFilterTitlePatterns"
                    value={(active.torrentFilter.titlePatterns ?? []).join("\n",)}
                    onChange={(e,) => {
                      const lines = e.target.value.split("\n",);
                      const result = lines.filter(Boolean,);
                      if (e.target.value.endsWith("\n",)) {
                        result.push("",);
                      }
                      updateTorrentFilter({ titlePatterns: result, },);
                    }}
                    className="font-mono text-sm bg-card border border-border rounded px-3 py-2 w-full min-h-[80px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green/50 resize-y"
                    placeholder={t("dashboard.torrentFilterTitlePatternsPlaceholder",)}
                    rows={4}
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torrentFilterTitlePatternsHint",)}</span>
                </div>

                <div>
                  <label htmlFor="torrentFilterFilenamePatterns" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.torrentFilterFilenamePatterns",)}</label>
                  <textarea
                    id="torrentFilterFilenamePatterns"
                    value={(active.torrentFilter.filenamePatterns ?? []).join("\n",)}
                    onChange={(e,) => {
                      const lines = e.target.value.split("\n",);
                      const result = lines.filter(Boolean,);
                      if (e.target.value.endsWith("\n",)) {
                        result.push("",);
                      }
                      updateTorrentFilter({ filenamePatterns: result, },);
                    }}
                    className="font-mono text-sm bg-card border border-border rounded px-3 py-2 w-full min-h-[80px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green/50 resize-y"
                    placeholder={t("dashboard.torrentFilterFilenamePatternsPlaceholder",)}
                    rows={4}
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torrentFilterFilenamePatternsHint",)}</span>
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.tmdbGroup",)}</legend>

          <div className="flex flex-col">
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={active.tmdb.enabled}
                onCheckedChange={(v: boolean,) => v != null && updateTmdb({ enabled: v, },)}
                className="size-4"
              />
              <span className="font-mono text-xs text-foreground uppercase">{t("dashboard.tmdbEnabled",)}</span>
            </label>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.tmdbEnabledHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.tmdb.enabled
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col gap-4 px-0.5">
                <div className="mt-4">
                  <label htmlFor="tmdbAccessToken" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.tmdbAccessToken",)}
                  </label>
                  <Input
                    id="tmdbAccessToken"
                    type="text"
                    value={active.tmdb.accessToken}
                    onChange={(e,) => updateTmdb({ accessToken: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.tmdbAccessTokenHint",)}</span>
                </div>

                <div>
                  <label htmlFor="tmdbRateLimit" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.tmdbRateLimit",)}
                  </label>
                  <Input
                    id="tmdbRateLimit"
                    type="number"
                    min={1}
                    max={40}
                    value={active.tmdb.rateLimit}
                    onChange={(e,) => {
                      const v = parseInt(e.target.value, 10,);
                      if (!isNaN(v,) && v >= 1 && v <= 40) {
                        updateTmdb({ rateLimit: v, },);
                      }
                    }}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.tmdbRateLimitHint",)}</span>
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <div className="flex items-center justify-between gap-3 pt-2">
          <div className="flex flex-col gap-1">
            <Button
              variant="default"
              size="sm"
              className="gap-1.5 font-mono text-xs bg-cyan/10 text-cyan border border-cyan/40 hover:bg-cyan/20 disabled:opacity-50"
              onClick={startReclassify}
              disabled={reclassifyBusy}
            >
              {reclassifyRunning ? (
                <>
                  <LoaderCircle className="size-3.5 animate-spin" />
                  {t("dashboard.reclassifying", { processed: reclassifyProcessed, total: reclassifyTotal, },)}
                </>
              ) : (
                <>
                  <RefreshCw className="size-3.5" />
                  {t("dashboard.reclassifyAll",)}
                </>
              )}
            </Button>
            {reindexRunning && (
              <span className="font-mono text-[10px] text-amber">{t("dashboard.reindexInProgressHint",)}</span>
            )}
          </div>

          <Button
            variant="default"
            size="sm"
            className="gap-1.5 font-mono text-xs bg-green/20 text-green border border-green/40 hover:bg-green/30"
            onClick={save}
            disabled={saving}
          >
            {saving ? (
              <>
                <LoaderCircle className="size-3.5 animate-spin" />
                {t("dashboard.saving",)}
              </>
            ) : (
              t("dashboard.save",)
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
