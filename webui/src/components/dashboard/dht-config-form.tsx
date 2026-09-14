import { useState, } from "react";
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
import type { ConfigQuery, DhtConfigInput, } from "@/lib/graphql/generated/graphql";

type DHTSection = ConfigQuery["config"]["dht"];

export function DHTConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<DHTSection | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.dht ?? null;
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

  function update(partial: Partial<DHTSection>,) {
    setForm({ ...active!, ...partial, } as DHTSection,);
  }

  function updateResponder(partial: Partial<DHTSection["responder"]>,) {
    setForm({ ...active!, responder: { ...active!.responder, ...partial, }, } as DHTSection,);
  }

  async function save() {
    if (saving || !original || !form) return;
    setSaving(true,);
    try {
      const patch: Record<string, unknown> = {};

      if (form.port !== original.port) patch.port = form.port;
      if (form.reseedBootstrapNodesInterval !== original.reseedBootstrapNodesInterval) {
        patch.reseedBootstrapNodesInterval = form.reseedBootstrapNodesInterval;
      }

      if (JSON.stringify(form.bootstrapNodes,) !== JSON.stringify(original.bootstrapNodes,)) {
        patch.bootstrapNodes = form.bootstrapNodes.filter(Boolean,);
      }

      const responderPatch: Record<string, unknown> = {};
      if (form.responder.enabled !== original.responder.enabled) responderPatch.enabled = form.responder.enabled;
      if (form.responder.globalRateLimit !== original.responder.globalRateLimit) {
        responderPatch.globalRateLimit = form.responder.globalRateLimit;
      }
      if (form.responder.perIPRateLimit !== original.responder.perIPRateLimit) {
        responderPatch.perIPRateLimit = form.responder.perIPRateLimit;
      }
      if (Object.keys(responderPatch,).length > 0) patch.responder = responderPatch;

      const requesterPatch: Record<string, unknown> = {};
      if (form.requester.requestLimit !== original.requester.requestLimit) {
        requesterPatch.requestLimit = form.requester.requestLimit;
      }
      if (form.requester.hashDiscoverLimit !== original.requester.hashDiscoverLimit) {
        requesterPatch.hashDiscoverLimit = form.requester.hashDiscoverLimit;
      }
      if (form.requester.rescrapeThreshold !== original.requester.rescrapeThreshold) {
        requesterPatch.rescrapeThreshold = form.requester.rescrapeThreshold;
      }
      if (Object.keys(requesterPatch,).length > 0) patch.requester = requesterPatch;

      const result = await updateConfig({ input: { dht: patch as DhtConfigInput, }, },);
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

  return (
    <Card className="border-border">
      <CardHeader className="pb-0">
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.dhtConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.dhtGeneral",)}</legend>

          <div>
            <label htmlFor="dhtPort" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.dhtPort",)}
            </label>
            <Input
              id="dhtPort"
              type="number"
              min={1}
              max={65535}
              value={active.port}
              onChange={(e,) => update({ port: parseInt(e.target.value, 10,) || 0, },)}
              className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">
              {t("dashboard.dhtPortHint",)} — {t("dashboard.restartRequired",)}
            </span>
          </div>

          <div>
            <label htmlFor="dhtBootstrapNodes" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.bootstrapNodes",)}
            </label>
            <textarea
              id="dhtBootstrapNodes"
              rows={4}
              value={(active.bootstrapNodes ?? []).join("\n",)}
              onChange={(e,) => {
                const lines = e.target.value.split("\n",).filter(Boolean,);
                if (e.target.value.endsWith("\n",)) {
                  lines.push("",);
                }
                update({ bootstrapNodes: lines, },);
              }}
              className="font-mono text-base md:text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.bootstrapNodesHint",)}</span>
          </div>

          <div>
            <label htmlFor="dhtRequestLimit" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.requesterRequestLimit",)}
            </label>
            <Input
              id="dhtRequestLimit"
              type="number"
              min={0}
              value={active.requester.requestLimit}
              onChange={(e,) => update({
                requester: { ...active.requester, requestLimit: parseInt(e.target.value, 10,) || 0, },
              } as unknown as Partial<DHTSection>,)}
              className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.requesterRequestLimitHint",)}</span>
          </div>

          <div>
            <label htmlFor="dhtHashDiscoverLimit" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.hashDiscoverLimit",)}
            </label>
            <Input
              id="dhtHashDiscoverLimit"
              type="number"
              min={1}
              value={active.requester.hashDiscoverLimit}
              onChange={(e,) => update({
                requester: { ...active.requester, hashDiscoverLimit: parseInt(e.target.value, 10,) || 0, },
              } as unknown as Partial<DHTSection>,)}
              className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.hashDiscoverLimitHint",)}</span>
          </div>

          <div>
            <label htmlFor="dhtRescrapeThreshold" className="font-mono text-xs text-foreground uppercase block mb-1.5">
              {t("dashboard.rescrapeThreshold",)}
            </label>
            <Input
              id="dhtRescrapeThreshold"
              type="number"
              min={1}
              value={active.requester.rescrapeThreshold}
              onChange={(e,) => update({
                requester: { ...active.requester, rescrapeThreshold: Number(e.target.value,), },
              } as unknown as Partial<DHTSection>,)}
              className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.rescrapeThresholdHint",)}</span>
          </div>

        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.responderGroup",)}</legend>

          <div className="flex flex-col">
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={active.responder.enabled}
                onCheckedChange={(v: boolean,) => updateResponder({ enabled: v, },)}
                className="size-4"
              />
              <span className="font-mono text-xs text-foreground uppercase">{t("dashboard.responderEnabled",)}</span>
            </label>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.responderEnabledHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.responder.enabled
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col px-0.5">
                <div className="mt-4">
                  <label htmlFor="globalRateLimit" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.responderGlobalRateLimit",)}
                  </label>
                  <Input
                    id="globalRateLimit"
                    type="number"
                    min={0}
                    value={active.responder.globalRateLimit}
                    onChange={(e,) =>
                      updateResponder({ globalRateLimit: parseInt(e.target.value, 10,) || 0, },)
                    }
                    className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.responderGlobalRateLimitHint",)}</span>
                </div>

                <div>
                  <label htmlFor="perIPRateLimit" className="font-mono text-xs text-foreground uppercase block mb-1.5">
                    {t("dashboard.responderPerIPRateLimit",)}
                  </label>
                  <Input
                    id="perIPRateLimit"
                    type="number"
                    min={0}
                    value={active.responder.perIPRateLimit}
                    onChange={(e,) =>
                      updateResponder({ perIPRateLimit: parseInt(e.target.value, 10,) || 0, },)
                    }
                    className="font-mono bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.responderPerIPRateLimitHint",)}</span>
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <div className="flex items-center justify-end gap-3 pt-2">
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
