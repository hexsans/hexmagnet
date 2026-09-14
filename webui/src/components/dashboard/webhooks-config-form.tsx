import { useState, useRef, } from "react";
import { useQuery, useMutation, } from "urql";
import { useTranslation, } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { Input, } from "@/components/ui/input";
import { Checkbox, } from "@/components/ui/checkbox";
import { Skeleton, } from "@/components/ui/skeleton";
import { LoaderCircle, } from "lucide-react";
import { Button, } from "@/components/ui/button";
import { toast, } from "sonner";
import { ConfigDocument, UpdateConfigDocument, } from "@/lib/graphql/generated/graphql";
import type { ConfigQuery, WebhooksConfigInput, } from "@/lib/graphql/generated/graphql";

type WebhooksSection = ConfigQuery["config"]["webhooks"];

const URL_PATTERN = /^https?:\/\/.+/i;

const CONTENT_TYPES = [
  "movie",
  "tv_show",
  "music",
  "ebook",
  "comic",
  "audiobook",
  "game",
  "software",
  "other",
  "unknown",
  "adult",
];

export function WebhooksConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<WebhooksSection | null>(null,);
  const [errors, setErrors,] = useState<Record<string, string | null>>({},);
  const validateDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.webhooks ?? null;
  const active = form ?? original;

  if (fetching || !active) {
    return (
      <Card className="border-border">
        <CardHeader>
          <Skeleton className="h-4 w-32" />
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

  function validate(cfg: WebhooksSection,): Record<string, string | null> {
    const errs: Record<string, string | null> = {};

    if (cfg.enabled) {
      if (cfg.urls.length === 0) errs.urls = t("dashboard.webhooksUrlsEmpty",);
      for (const u of cfg.urls) {
        if (!URL_PATTERN.test(u,)) {
          errs.urls = t("dashboard.webhooksUrlInvalid",);
          break;
        }
      }
    }

    if (cfg.timeout < 1) errs.timeout = t("dashboard.errorRange", { min: 1, max: 3600, },);
    if (cfg.maxRetries > 10) errs.maxRetries = t("dashboard.errorRange", { min: 0, max: 10, },);
    if (cfg.queueSize < 1) errs.queueSize = t("dashboard.errorRange", { min: 1, max: 100000, },);

    if (cfg.baseUrl !== "" && !URL_PATTERN.test(cfg.baseUrl,)) {
      errs.baseUrl = t("dashboard.webhooksUrlInvalid",);
    }

    if (cfg.events.length === 0) errs.events = t("dashboard.webhooksEventsEmpty",);

    for (const h of cfg.headers ?? []) {
      if (h.key.trim() === "" && h.value.trim() !== "") {
        errs.headers = t("dashboard.webhooksHeaderKeyEmpty",);
        break;
      }
    }

    return errs;
  }

  function normalizedHeaders(headers: WebhooksSection["headers"] | undefined,): { key: string; value: string }[] {
    return (headers ?? [])
      .filter((h,) => h.key.trim() !== "" || h.value.trim() !== "",)
      .map((h,) => ({ key: h.key.trim(), value: h.value, }),);
  }

  function mergeDeep(base: WebhooksSection, patch: DeepPartial<WebhooksSection>,): WebhooksSection {
    return {
      enabled: patch.enabled ?? base.enabled,
      urls: patch.urls ?? base.urls,
      events: patch.events ?? base.events,
      categories: patch.categories ?? base.categories,
      titlePatterns: patch.titlePatterns ?? base.titlePatterns,
      filenamePatterns: patch.filenamePatterns ?? base.filenamePatterns,
      timeout: patch.timeout ?? base.timeout,
      maxRetries: patch.maxRetries ?? base.maxRetries,
      baseUrl: patch.baseUrl ?? base.baseUrl,
      headers: (patch.headers as WebhooksSection["headers"] | undefined) ?? base.headers,
      queueSize: patch.queueSize ?? base.queueSize,
    };
  }

  function update(partial: DeepPartial<WebhooksSection>,) {
    const next = mergeDeep(active!, partial,);
    setForm(next,);
    if (validateDebounceRef.current) clearTimeout(validateDebounceRef.current,);
    validateDebounceRef.current = setTimeout(() => {
      setErrors(validate(next,),);
    }, 300,);
  }

  async function save() {
    if (!form || saving || !original) return;
    const errs = validate(form,);
    setErrors(errs,);
    if (Object.values(errs,).some(Boolean,)) return;
    setSaving(true,);
    try {
      const patch: Record<string, unknown> = {};
      if (form.enabled !== original.enabled) patch.enabled = form.enabled;
      if (JSON.stringify(form.urls,) !== JSON.stringify(original.urls,)) {
        patch.urls = form.urls.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (JSON.stringify(form.events,) !== JSON.stringify(original.events,)) {
        patch.events = form.events.filter(Boolean,);
      }
      if (JSON.stringify(form.categories,) !== JSON.stringify(original.categories,)) {
        patch.categories = form.categories.filter(Boolean,);
      }
      if (JSON.stringify(form.titlePatterns,) !== JSON.stringify(original.titlePatterns,)) {
        patch.titlePatterns = form.titlePatterns.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (JSON.stringify(form.filenamePatterns,) !== JSON.stringify(original.filenamePatterns,)) {
        patch.filenamePatterns = form.filenamePatterns.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (form.timeout !== original.timeout) patch.timeout = form.timeout;
      if (form.maxRetries !== original.maxRetries) patch.maxRetries = form.maxRetries;
      if (form.baseUrl !== original.baseUrl) patch.baseUrl = form.baseUrl;
      if (JSON.stringify(normalizedHeaders(form.headers,),) !== JSON.stringify(normalizedHeaders(original.headers,),)) {
        patch.headers = normalizedHeaders(form.headers,);
      }
      if (form.queueSize !== original.queueSize) patch.queueSize = form.queueSize;

      const result = await updateConfig({ input: { webhooks: patch as WebhooksConfigInput, }, },);
      if (result.error) {
        toast.error(result.error.message,);
        return;
      }
      setForm(null,);
      reexecQuery({ requestPolicy: "network-only", },);
      setErrors({},);
      toast.success(t("dashboard.saved",),);
    } catch {
      toast.error(t("dashboard.saveFailed",),);
    } finally {
      setSaving(false,);
    }
  }

  const hasErrors = Object.values(errors,).some(Boolean,);

  return (
    <Card className="border-border">
      <CardHeader className="pb-0">
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.webhooksConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.webhooksGroup",)}</legend>

          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <Checkbox
              id="webhooksEnabled"
              checked={active.enabled}
              onCheckedChange={(v,) => update({ enabled: v === true, },)}
            />
            <label htmlFor="webhooksEnabled" className="font-mono text-xs text-foreground uppercase cursor-pointer">
              {t("dashboard.webhooksEnabled",)}
            </label>
            <span className="font-mono text-[10px] text-muted-foreground/60 block w-full sm:mt-0.5 sm:w-auto">{t("dashboard.webhooksEnabledHint",)}</span>
          </div>

          <div>
            <label htmlFor="webhooksUrls" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksUrls",)}
            </label>
            <textarea
              id="webhooksUrls"
              rows={4}
              value={(active.urls ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ urls: lines, },);
              }}
              placeholder="https://example.com/hook"
              className={`font-mono text-base md:text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y ${errors.urls ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksUrlsHint",)}</span>
            {errors.urls && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.urls}</span>}
          </div>

          <div>
            <label htmlFor="webhooksEvents" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksEvents",)}
            </label>
            <div className="flex flex-wrap gap-3">
              <label className="flex items-center gap-2 cursor-pointer">
                <Checkbox
                  id="webhooksEventsClassified"
                  checked={active.events.includes("classified",)}
                  onCheckedChange={(v,) => {
                    const next = v === true
                      ? [...new Set([...active.events, "classified",],),]
                      : active.events.filter((e,) => e !== "classified",);
                    update({ events: next, },);
                  }}
                />
                <span className="font-mono text-xs text-foreground">{t("dashboard.webhooksEventClassified",)}</span>
              </label>
            </div>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksEventsHint",)}</span>
            {errors.events && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.events}</span>}
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="webhooksTimeout" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                {t("dashboard.webhooksTimeout",)}
              </label>
              <Input
                id="webhooksTimeout"
                type="number"
                min={1}
                value={active.timeout}
                onChange={(e,) => update({ timeout: parseInt(e.target.value, 10,) || 0, },)}
                className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.timeout ? "border-red" : ""}`}
              />
              <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksTimeoutHint",)}</span>
              {errors.timeout && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.timeout}</span>}
            </div>

            <div>
              <label htmlFor="webhooksMaxRetries" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                {t("dashboard.webhooksMaxRetries",)}
              </label>
              <Input
                id="webhooksMaxRetries"
                type="number"
                min={0}
                max={10}
                value={active.maxRetries}
                onChange={(e,) => update({ maxRetries: parseInt(e.target.value, 10,) || 0, },)}
                className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.maxRetries ? "border-red" : ""}`}
              />
              <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksMaxRetriesHint",)}</span>
              {errors.maxRetries && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.maxRetries}</span>}
            </div>

            <div>
              <label htmlFor="webhooksQueueSize" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                {t("dashboard.webhooksQueueSize",)}
              </label>
              <Input
                id="webhooksQueueSize"
                type="number"
                min={1}
                value={active.queueSize}
                onChange={(e,) => update({ queueSize: parseInt(e.target.value, 10,) || 0, },)}
                className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.queueSize ? "border-red" : ""}`}
              />
              <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksQueueSizeHint",)}</span>
              {errors.queueSize && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.queueSize}</span>}
            </div>
          </div>

          <div>
            <label htmlFor="webhooksBaseUrl" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksBaseUrl",)}
            </label>
            <Input
              id="webhooksBaseUrl"
              type="text"
              value={active.baseUrl}
              onChange={(e,) => update({ baseUrl: e.target.value, },)}
              placeholder="http://localhost:3333"
              className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.baseUrl ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksBaseUrlHint",)}</span>
            {errors.baseUrl && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.baseUrl}</span>}
          </div>

          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="font-mono text-xs text-foreground uppercase">{t("dashboard.webhooksHeaders",)}</label>
              <Button
                variant="outline"
                size="sm"
                className="h-6 px-2 font-mono text-[10px] uppercase"
                onClick={() => update({ headers: [...normalizedHeaders(active.headers,), { key: "", value: "", },], },)}
              >
                {t("dashboard.webhooksAddHeader",)}
              </Button>
            </div>
            <div className="flex flex-col gap-2">
              {normalizedHeaders(active.headers,).map((h, i,) => {
                const rows = normalizedHeaders(active.headers,);
                return (
                  <div key={i} className="flex flex-wrap items-center gap-2">
                    <Input
                      type="text"
                      value={h.key}
                      placeholder="Header"
                      onChange={(e,) => {
                        const next = [...rows,];
                        next[i] = { key: e.target.value, value: h.value, };
                        update({ headers: next, },);
                      }}
                      className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 w-2/5"
                    />
                    <Input
                      type="text"
                      value={h.value}
                      placeholder="Value"
                      onChange={(e,) => {
                        const next = [...rows,];
                        next[i] = { key: h.key, value: e.target.value, };
                        update({ headers: next, },);
                      }}
                      className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 flex-1"
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 w-8 p-0 text-muted-foreground hover:text-red"
                      onClick={() => {
                        const next = rows.filter((_, j,) => j !== i,);
                        update({ headers: next, },);
                      }}
                    >
                      ×
                    </Button>
                  </div>
                );
              },)}
            </div>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksHeadersHint",)}</span>
            {errors.headers && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.headers}</span>}
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.webhooksFilterGroup",)}</legend>

          <div>
            <label htmlFor="webhooksCategories" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksCategories",)}
            </label>
            <div className="flex flex-wrap gap-3">
              {CONTENT_TYPES.map((ct,) => (
                <label key={ct} className="flex items-center gap-2 cursor-pointer">
                  <Checkbox
                    id={`webhooksCategories${ct}`}
                    checked={(active.categories ?? []).includes(ct,)}
                    onCheckedChange={(v,) => {
                      const cur = active.categories ?? [];
                      const next = v === true
                        ? [...new Set([...cur, ct,],),]
                        : cur.filter((c,) => c !== ct,);
                      update({ categories: next, },);
                    }}
                  />
                  <span className="font-mono text-xs text-foreground">{ct}</span>
                </label>
              ),)}
            </div>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksCategoriesHint",)}</span>
          </div>

          <div>
            <label htmlFor="webhooksTitlePatterns" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksTitleRegex",)}
            </label>
            <textarea
              id="webhooksTitlePatterns"
              rows={3}
              value={(active.titlePatterns ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ titlePatterns: lines, },);
              }}
              placeholder={"FLAC|2160p|4K"}
              className="font-mono text-base md:text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksTitleRegexHint",)}</span>
          </div>

          <div>
            <label htmlFor="webhooksFilenamePatterns" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.webhooksFilenameRegex",)}
            </label>
            <textarea
              id="webhooksFilenamePatterns"
              rows={3}
              value={(active.filenamePatterns ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ filenamePatterns: lines, },);
              }}
              placeholder={String.raw`\.flac$|\.mkv$`}
              className="font-mono text-base md:text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.webhooksFilenameRegexHint",)}</span>
          </div>
        </fieldset>

        <div className="flex items-center justify-end gap-3 pt-2">
          <Button
            variant="default"
            size="sm"
            className="gap-1.5 font-mono text-xs bg-green/20 text-green border border-green/40 hover:bg-green/30"
            onClick={save}
            disabled={saving || hasErrors}
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

type DeepPartial<T,> = T extends Array<infer U>
  ? Array<DeepPartial<U>>
  : T extends object
    ? { [P in keyof T]?: DeepPartial<T[P]> }
    : T;