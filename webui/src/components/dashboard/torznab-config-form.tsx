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
import type { ConfigQuery, TorznabConfigInput, } from "@/lib/graphql/generated/graphql";

const CATEGORY_PRESETS = ["*", "2000", "3000", "5000", "7000", "8000",] as const;

type TorznabSection = ConfigQuery["config"]["torznab"];

export function TorznabConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<TorznabSection | null>(null,);
  const [errors, setErrors,] = useState<Record<string, string | null>>({},);
  const validateDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.torznab ?? null;
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

  function validate(cfg: TorznabSection,): Record<string, string | null> {
    const errs: Record<string, string | null> = {};
    if (cfg.path.trim() === "") errs.path = t("dashboard.errorRequired",);
    if (cfg.maxResults < 1) errs.maxResults = t("dashboard.errorRange", { min: 1, max: 100000, },);
    if (cfg.categories.length === 0) errs.categories = t("dashboard.torznabCategoriesEmpty",);
    return errs;
  }

  function mergeDeep(base: TorznabSection, patch: DeepPartial<TorznabSection>,): TorznabSection {
    return {
      enabled: patch.enabled ?? base.enabled,
      apiKey: patch.apiKey ?? base.apiKey,
      path: patch.path ?? base.path,
      maxResults: patch.maxResults ?? base.maxResults,
      categories: patch.categories ?? base.categories,
      trustProxyHeaders: patch.trustProxyHeaders ?? base.trustProxyHeaders,
    };
  }

  function update(partial: DeepPartial<TorznabSection>,) {
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
      if (form.apiKey !== original.apiKey) patch.apiKey = form.apiKey;
      if (form.path !== original.path) patch.path = form.path;
      if (form.maxResults !== original.maxResults) patch.maxResults = form.maxResults;
      if (form.trustProxyHeaders !== original.trustProxyHeaders) patch.trustProxyHeaders = form.trustProxyHeaders;
      if (JSON.stringify(form.categories,) !== JSON.stringify(original.categories,)) {
        patch.categories = form.categories.filter(Boolean,);
      }

      const result = await updateConfig({ input: { torznab: patch as TorznabConfigInput, }, },);
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
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.torznabConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.torznabGroup",)}</legend>

          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <Checkbox
              id="torznabEnabled"
              checked={active.enabled}
              onCheckedChange={(v,) => update({ enabled: v === true, },)}
            />
            <label htmlFor="torznabEnabled" className="font-mono text-xs text-foreground uppercase cursor-pointer">
              {t("dashboard.torznabEnabled",)}
            </label>
            <span className="font-mono text-[10px] text-muted-foreground/60 block w-full sm:mt-0.5 sm:w-auto">{t("dashboard.torznabEnabledHint",)}</span>
          </div>

          <div>
            <label htmlFor="torznabApiKey" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.torznabApiKey",)}
            </label>
            <Input
              id="torznabApiKey"
              type="password"
              value={active.apiKey}
              onChange={(e,) => update({ apiKey: e.target.value, },)}
              placeholder={t("dashboard.torznabApiKeyHint",)}
              className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torznabApiKeyHint",)}</span>
          </div>

          <div>
            <label htmlFor="torznabPath" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.torznabPath",)}
            </label>
            <Input
              id="torznabPath"
              type="text"
              value={active.path}
              onChange={(e,) => update({ path: e.target.value, },)}
              placeholder="/torznab"
              className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.path ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">
              {t("dashboard.torznabPathHint",)} — {t("dashboard.restartRequired",)}
            </span>
            {errors.path && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.path}</span>}
          </div>

          <div>
            <label htmlFor="torznabMaxResults" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.torznabMaxResults",)}
            </label>
            <Input
              id="torznabMaxResults"
              type="number"
              min={1}
              value={active.maxResults}
              onChange={(e,) => update({ maxResults: parseInt(e.target.value, 10,) || 0, },)}
              className={`font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.maxResults ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torznabMaxResultsHint",)}</span>
            {errors.maxResults && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.maxResults}</span>}
          </div>

          <div>
            <label htmlFor="torznabCategories" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.torznabCategories",)}
            </label>
            <textarea
              id="torznabCategories"
              rows={3}
              value={(active.categories ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ categories: lines, },);
              }}
              placeholder={t("dashboard.torznabCategoriesPlaceholder",)}
              className="font-mono text-base md:text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torznabCategoriesHint",)}</span>
            {errors.categories && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.categories}</span>}

            <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-3">
              <Checkbox
                id="torznabTrustProxy"
                checked={active.trustProxyHeaders}
                onCheckedChange={(v,) => update({ trustProxyHeaders: v === true, },)}
              />
              <label htmlFor="torznabTrustProxy" className="font-mono text-xs text-foreground uppercase cursor-pointer">
                {t("dashboard.torznabTrustProxyHeaders",)}
              </label>
              <span className="font-mono text-[10px] text-muted-foreground/60 block w-full sm:mt-0.5 sm:w-auto">{t("dashboard.torznabTrustProxyHeadersHint",)}</span>
            </div>

            <div className="flex flex-wrap gap-1.5 mt-2">
              {CATEGORY_PRESETS.map((c,) => (
                <Button
                  key={c}
                  type="button"
                  size="sm"
                  variant="outline"
                  className="h-6 px-2 font-mono text-[10px] border-border text-muted-foreground hover:text-foreground"
                  onClick={() => update({ categories: [c,], },)}
                >
                  {c === "*" ? t("dashboard.torznabCatAll",) : c}
                </Button>
              ),)}
            </div>
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