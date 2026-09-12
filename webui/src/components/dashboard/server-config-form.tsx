import { useState, useRef, } from "react";
import { useQuery, useMutation, } from "urql";
import { useTranslation, } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { Input, } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, } from "@/components/ui/select";
import { Skeleton, } from "@/components/ui/skeleton";
import { LoaderCircle, } from "lucide-react";
import { Button, } from "@/components/ui/button";
import { toast, } from "sonner";
import { ConfigDocument, UpdateConfigDocument, } from "@/lib/graphql/generated/graphql";
import type { ConfigQuery, ServerConfigInput, } from "@/lib/graphql/generated/graphql";

const LOG_LEVELS = ["debug", "info", "warn", "error",] as const;
const FILE_LOG_LEVELS = ["off", ...LOG_LEVELS,] as const;
const LOG_FORMATS = ["text", "json",] as const;

type ServerSection = ConfigQuery["config"]["server"];

export function ServerConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<ServerSection | null>(null,);
  const [errors, setErrors,] = useState<Record<string, string | null>>({},);
  const validateDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.server ?? null;
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

  function validate(cfg: ServerSection,): Record<string, string | null> {
    const errs: Record<string, string | null> = {};
    if (cfg.port < 1 || cfg.port > 65535) errs.port = t("dashboard.errorRange", { min: 1, max: 65535, },);
    return errs;
  }

  function mergeDeep(base: ServerSection, patch: DeepPartial<ServerSection>,): ServerSection {
    return {
      ip: patch.ip ?? base.ip,
      port: patch.port ?? base.port,
      embedTrackers: patch.embedTrackers?.filter((x,): x is string => x != null,) ?? base.embedTrackers,
      torrentFilePath: patch.torrentFilePath ?? base.torrentFilePath,
      log: {
        consoleLevel: patch.log?.consoleLevel ?? base.log.consoleLevel,
        fileOutputLevel: patch.log?.fileOutputLevel ?? base.log.fileOutputLevel,
        fileRotator: {
          path: patch.log?.fileRotator?.path ?? base.log.fileRotator.path,
          maxBackups: patch.log?.fileRotator?.maxBackups ?? base.log.fileRotator.maxBackups,
          maxSizeMB: patch.log?.fileRotator?.maxSizeMB ?? base.log.fileRotator.maxSizeMB,
          format: patch.log?.fileRotator?.format ?? base.log.fileRotator.format,
        },
      },
    };
  }

  function update(partial: DeepPartial<ServerSection>,) {
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
      if (form.ip !== original.ip) patch.ip = form.ip;
      if (form.port !== original.port) patch.port = form.port;
      const logPatch: Record<string, unknown> = {};
      if (form.log.consoleLevel !== original.log.consoleLevel) logPatch.consoleLevel = form.log.consoleLevel;
      if (form.log.fileOutputLevel !== original.log.fileOutputLevel) logPatch.fileOutputLevel = form.log.fileOutputLevel;
      const rotatorPatch: Record<string, unknown> = {};
      if (form.log.fileRotator.path !== original.log.fileRotator.path) rotatorPatch.path = form.log.fileRotator.path;
      if (form.log.fileRotator.maxBackups !== original.log.fileRotator.maxBackups) {
        rotatorPatch.maxBackups = form.log.fileRotator.maxBackups;
      }
      if (form.log.fileRotator.maxSizeMB !== original.log.fileRotator.maxSizeMB) {
        rotatorPatch.maxSizeMB = form.log.fileRotator.maxSizeMB;
      }
      if (form.log.fileRotator.format !== original.log.fileRotator.format) rotatorPatch.format = form.log.fileRotator.format;
      if (Object.keys(rotatorPatch,).length > 0) logPatch.fileRotator = rotatorPatch;
      if (Object.keys(logPatch,).length > 0) patch.log = logPatch;
      if (JSON.stringify(form.embedTrackers,) !== JSON.stringify(original.embedTrackers,)) {
        patch.embedTrackers = form.embedTrackers.filter(Boolean,);
      }
      if (form.torrentFilePath !== original.torrentFilePath) patch.torrentFilePath = form.torrentFilePath;

      const result = await updateConfig({ input: { server: patch as ServerConfigInput, }, },);
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
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.generalConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.serverGroup",)}</legend>

          <div>
            <label htmlFor="ip" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.ip",)}
            </label>
            <Input
              id="ip"
              type="text"
              value={active.ip}
              onChange={(e,) => update({ ip: e.target.value, },)}
              placeholder={t("dashboard.ipEmptyHint",)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">
              {active.ip === ""
                ? t("dashboard.ipEmptyHint",)
                : active.ip === "0.0.0.0"
                  ? t("dashboard.ipV4Hint",)
                  : active.ip === "::"
                    ? t("dashboard.ipV6Hint",)
                    : null}
              {" — "}{t("dashboard.restartRequired",)}
            </span>
          </div>

          <div>
            <label htmlFor="port" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.port",)}
            </label>
            <Input
              id="port"
              type="number"
              value={active.port}
              onChange={(e,) => update({ port: parseInt(e.target.value, 10,), },)}
              className={`font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.port ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.restartRequired",)}</span>
            {errors.port && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.port}</span>}
          </div>

          <div>
            <label htmlFor="torrentFilePath" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.torrentFilePath",)}
            </label>
            <Input
              id="torrentFilePath"
              type="text"
              value={active.torrentFilePath}
              onChange={(e,) => update({ torrentFilePath: e.target.value, },)}
              placeholder={t("dashboard.torrentFilePathHint",)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.torrentFilePathHint",)}</span>
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.logging",)}</legend>

          <div>
            <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.consoleLevel",)}</label>
            <Select value={active.log.consoleLevel} onValueChange={(v,) => v != null && update({ log: { consoleLevel: v, }, },)}>
              <SelectTrigger className="w-full font-mono text-sm border-border">
                <span className="flex-1 text-left truncate">{{
                  debug: t("dashboard.logLevelDebug",),
                  info: t("dashboard.logLevelInfo",),
                  warn: t("dashboard.logLevelWarn",),
                  error: t("dashboard.logLevelError",),
                }[active.log.consoleLevel]}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
                {LOG_LEVELS.map((l,) => (
                  <SelectItem key={l} value={l}>{t("dashboard.logLevel" + l.charAt(0,).toUpperCase() + l.slice(1,),)}</SelectItem>
                ),)}
              </SelectContent>
            </Select>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.consoleLevelHint",)}</span>
          </div>

          <div>
            <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.fileOutputLevel",)}</label>
            <Select value={active.log.fileOutputLevel} onValueChange={(v,) => v != null && update({ log: { fileOutputLevel: v, }, },)}>
              <SelectTrigger className="w-full font-mono text-sm border-border">
                <span className="flex-1 text-left truncate">{{
                  off: t("dashboard.fileOutputLevelOff",),
                  debug: t("dashboard.logLevelDebug",),
                  info: t("dashboard.logLevelInfo",),
                  warn: t("dashboard.logLevelWarn",),
                  error: t("dashboard.logLevelError",),
                }[active.log.fileOutputLevel] ?? t("dashboard.fileOutputLevelOff",)}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
                {FILE_LOG_LEVELS.map((l,) => (
                  <SelectItem key={l} value={l}>{l === "off" ? t("dashboard.fileOutputLevelOff",) : t("dashboard.logLevel" + l.charAt(0,).toUpperCase() + l.slice(1,),)}</SelectItem>
                ),)}
              </SelectContent>
            </Select>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.fileOutputLevelHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.log.fileOutputLevel !== "off"
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col gap-4 px-0.5">
                <div className="mt-4">
                  <label htmlFor="logPath" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.logPath",)}</label>
                  <Input
                    id="logPath"
                    type="text"
                    value={active.log.fileRotator.path}
                    onChange={(e,) => update({ log: { fileRotator: { path: e.target.value, }, }, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.logPathHint",)}</span>
                </div>

                <div>
                  <label htmlFor="maxLogFileCount" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.maxLogFileCount",)}</label>
                  <Input
                    id="maxLogFileCount"
                    type="number"
                    min={0}
                    value={active.log.fileRotator.maxBackups}
                    onChange={(e,) => update({ log: { fileRotator: { maxBackups: parseInt(e.target.value, 10,) || 0, }, }, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.maxLogFileCountHint",)}</span>
                </div>

                <div>
                  <label htmlFor="maxLogFileSize" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.maxLogFileSize",)}</label>
                  <Input
                    id="maxLogFileSize"
                    type="number"
                    min={0}
                    value={active.log.fileRotator.maxSizeMB}
                    onChange={(e,) => update({ log: { fileRotator: { maxSizeMB: parseInt(e.target.value, 10,) || 0, }, }, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.maxLogFileSizeHint",)}</span>
                </div>

                <div>
                  <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.logFormat",)}</label>
                  <Select value={active.log.fileRotator.format} onValueChange={(v,) => v != null && update({
                    log: { fileRotator: { format: v, }, },
                  },)}>
                    <SelectTrigger className="w-full font-mono text-sm border-border">
                      <span className="flex-1 text-left truncate">{{
                        text: t("dashboard.logFormatText",),
                        json: t("dashboard.logFormatJson",),
                      }[active.log.fileRotator.format]}</span>
                    </SelectTrigger>
                    <SelectContent className="font-mono text-sm">
                      {LOG_FORMATS.map((f,) => (
                        <SelectItem key={f} value={f}>{t("dashboard.logFormat" + f.charAt(0,).toUpperCase() + f.slice(1,),)}</SelectItem>
                      ),)}
                    </SelectContent>
                  </Select>
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.logFormatHint",)}</span>
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.otherGroup",)}</legend>

          <div>
            <label htmlFor="embedTrackers" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.embedTrackers",)}
            </label>
            <textarea
              id="embedTrackers"
              rows={3}
              value={(active.embedTrackers ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ embedTrackers: lines, },);
              }}
              className="font-mono text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embedTrackersHint",)}</span>
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

type DeepPartial<T,> = T extends object
  ? { [P in keyof T]?: DeepPartial<T[P]> }
  : T;
