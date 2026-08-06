import { useState, useRef, useEffect, } from "react";
import { useQuery, useMutation, useClient, } from "urql";
import { useTranslation, } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle, } from "@/components/ui/card";
import { Input, } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, } from "@/components/ui/select";
import { Skeleton, } from "@/components/ui/skeleton";
import { Checkbox, } from "@/components/ui/checkbox";
import { LoaderCircle, } from "lucide-react";
import { Button, } from "@/components/ui/button";
import { toast, } from "sonner";
import { ConfigDocument, UpdateConfigDocument, ReindexElasticsearchDocument, ReindexStatusDocument, } from "@/lib/graphql/generated/graphql";
import type { ConfigQuery, StorageConfigInput, } from "@/lib/graphql/generated/graphql";

type StorageSection = ConfigQuery["config"]["storage"];

export function StorageConfigForm() {
  const { t, } = useTranslation();
  const [form, setForm,] = useState<StorageSection | null>(null,);
  const [errors, setErrors,] = useState<Record<string, string | null>>({},);
  const validateDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null,);
  const [saving, setSaving,] = useState(false,);

  const [{ data, fetching, }, reexecQuery,] = useQuery({ query: ConfigDocument, },);
  const [, updateConfig,] = useMutation(UpdateConfigDocument,);

  const original = data?.config?.storage ?? null;
  const active = form ?? original;

  // Reindex state
  const [reindexing, setReindexing,] = useState(false,);
  const [reindexTotal, setReindexTotal,] = useState(0,);
  const [reindexIndexed, setReindexIndexed,] = useState(0,);
  const [, reindexMutation,] = useMutation(ReindexElasticsearchDocument,);
  const client = useClient();

  function startReindex() {
    reindexMutation({},).then((result,) => {
      const p = result.data?.torrent?.reindexToElasticsearch;
      if (!p) return;
      if (p.error) { toast.error(p.error,); return; }
      if (p.done) { toast.success(t("dashboard.reindexDone", { count: p.indexed, },),); return; }
      setReindexTotal(p.total,);
      setReindexIndexed(p.indexed,);
      setReindexing(true,);
    },);
  }

  // Detect if a reindex is already running on page load
  useEffect(() => {
    client.query(ReindexStatusDocument, {}, { requestPolicy: "network-only", },).toPromise().then((result,) => {
      const p = result.data?.reindexStatus;
      if (!p || !p.running) return;
      if (p.done) return;
      setReindexTotal(p.total,);
      setReindexIndexed(p.indexed,);
      setReindexing(true,);
    },);
  }, [client,],);

  // Poll reindex progress while active, stop when done
  useEffect(() => {
    if (!reindexing) return;

    let cancelled = false;

    const poll = () => {
      client.query(ReindexStatusDocument, {}, { requestPolicy: "network-only", },).toPromise().then((result,) => {
        if (cancelled) return;
        const p = result.data?.reindexStatus;
        if (!p) return;
        setReindexTotal(p.total,);
        setReindexIndexed(p.indexed,);
        if (p.error) { toast.error(p.error,); clearInterval(intervalId,); setReindexing(false,); return; }
        if (p.done) {
          clearInterval(intervalId,);
          if (p.indexed > 0) toast.success(t("dashboard.reindexDone", { count: p.indexed, },),);
          setReindexing(false,);
          return;
        }
        setReindexing(true,);
      },);
    };

    const intervalId = setInterval(poll, 1000,);
    poll();

    return () => {
      cancelled = true;
      clearInterval(intervalId,);
    };
  }, [reindexing, t, client,],);

  if (fetching || !active) {
    return (
      <Card className="border-border">
        <CardHeader>
          <Skeleton className="h-4 w-32" />
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            {Array.from({ length: 6, },).map((_, i,) => (
              <Skeleton key={i} className="h-10 w-full" />
            ),)}
          </div>
        </CardContent>
      </Card>
    );
  }

  function validate(cfg: StorageSection,): Record<string, string | null> {
    const errs: Record<string, string | null> = {};
    if (cfg.postgres.maxConnections < 1) errs.maxConnections = t("dashboard.errorMinValue", { count: 1, },);
    if (cfg.queue.backend === "kafka" && !cfg.queue.kafka.brokers.length) errs.brokers = t("dashboard.errorRequired",);
    if (cfg.search.backend === "elasticsearch" && !cfg.search.elasticsearch.addresses.length) errs.addresses = t("dashboard.errorRequired",);
    if (cfg.search.backend === "elasticsearch" && !cfg.search.elasticsearch.embedding?.endpoint) errs.embeddingEndpoint = t("dashboard.errorRequired",);
    if (cfg.search.backend === "elasticsearch" && !cfg.search.elasticsearch.embedding?.model) errs.embeddingModel = t("dashboard.errorRequired",);
    if (cfg.search.backend === "elasticsearch" && (cfg.search.elasticsearch.embedding?.dimensions ?? 0) < 1) errs.embeddingDimensions = t("dashboard.errorMinValue", { count: 1, },);
    return errs;
  }

  function update(partial: Partial<StorageSection>,) {
    const next = { ...active!, ...partial, } as StorageSection;
    setForm(next,);
    if (validateDebounceRef.current) clearTimeout(validateDebounceRef.current,);
    validateDebounceRef.current = setTimeout(() => {
      setErrors(validate(next,),);
    }, 300,);
  }

  function updatePostgres(partial: Partial<StorageSection["postgres"]>,) {
    update({ postgres: { ...active!.postgres, ...partial, }, } as Partial<StorageSection>,);
  }

  function updateKafka(partial: Partial<StorageSection["queue"]["kafka"]>,) {
    update({ queue: { ...active!.queue, kafka: { ...active!.queue.kafka, ...partial, }, }, } as unknown as Partial<StorageSection>,);
  }

  function updateElasticsearch(partial: Partial<StorageSection["search"]["elasticsearch"]>,) {
    update({
      search: { ...active!.search, elasticsearch: { ...active!.search.elasticsearch, ...partial, }, },
    } as unknown as Partial<StorageSection>,);
  }

  function updateElasticsearchEmbedding(partial: Partial<StorageSection["search"]["elasticsearch"]["embedding"]>,) {
    updateElasticsearch({ embedding: { ...active!.search.elasticsearch.embedding, ...partial, }, } as Partial<StorageSection["search"]["elasticsearch"]>,);
  }

  function updateSearch(partial: Partial<StorageSection["search"]>,) {
    update({ search: { ...active!.search, ...partial, }, } as unknown as Partial<StorageSection>,);
  }

  function updateQueue(partial: Partial<StorageSection["queue"]>,) {
    update({ queue: { ...active!.queue, ...partial, }, } as unknown as Partial<StorageSection>,);
  }

  async function save() {
    if (!form || saving || !original) return;
    const errs = validate(form,);
    setErrors(errs,);
    if (Object.values(errs,).some(Boolean,)) return;
    setSaving(true,);
    try {
      const pgPatch: Record<string, unknown> = {};
      if (form.postgres.host !== original.postgres.host) pgPatch.host = form.postgres.host;
      if (form.postgres.port !== original.postgres.port) pgPatch.port = form.postgres.port;
      if (form.postgres.username !== original.postgres.username) pgPatch.username = form.postgres.username;
      if (form.postgres.database !== original.postgres.database) pgPatch.database = form.postgres.database;
      if (form.postgres.password !== original.postgres.password) pgPatch.password = form.postgres.password;
      if (form.postgres.sslMode !== original.postgres.sslMode) pgPatch.sslMode = form.postgres.sslMode;
      if (form.postgres.connectionTimeout !== original.postgres.connectionTimeout) {
        pgPatch.connectionTimeout = form.postgres.connectionTimeout;
      }
      if (form.postgres.sslCertPath !== original.postgres.sslCertPath) pgPatch.sslCertPath = form.postgres.sslCertPath;
      if (form.postgres.sslKeyPath !== original.postgres.sslKeyPath) pgPatch.sslKeyPath = form.postgres.sslKeyPath;
      if (form.postgres.sslRootCertPath !== original.postgres.sslRootCertPath) pgPatch.sslRootCertPath = form.postgres.sslRootCertPath;
      if (form.postgres.maxConnections !== original.postgres.maxConnections) pgPatch.maxConnections = form.postgres.maxConnections;

      const searchPatch: Record<string, unknown> = {};
      if (form.search.backend !== original.search.backend) searchPatch.backend = form.search.backend;
      const esPatch: Record<string, unknown> = {};
      if (JSON.stringify(form.search.elasticsearch.addresses,) !== JSON.stringify(original.search.elasticsearch.addresses,)) {
        esPatch.addresses = form.search.elasticsearch.addresses.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (form.search.elasticsearch.embedding && original.search.elasticsearch.embedding) {
        const embPatch: Record<string, unknown> = {};
        if (form.search.elasticsearch.embedding.endpoint !== original.search.elasticsearch.embedding.endpoint) {
          embPatch.endpoint = (form.search.elasticsearch.embedding.endpoint ?? "").trim();
        }
        if (form.search.elasticsearch.embedding.apikey !== original.search.elasticsearch.embedding.apikey) {
          embPatch.apikey = form.search.elasticsearch.embedding.apikey ?? "";
        }
        if (form.search.elasticsearch.embedding.model !== original.search.elasticsearch.embedding.model) {
          embPatch.model = (form.search.elasticsearch.embedding.model ?? "").trim();
        }
        if (form.search.elasticsearch.embedding.dimensions !== original.search.elasticsearch.embedding.dimensions) {
          embPatch.dimensions = form.search.elasticsearch.embedding.dimensions;
        }
        if (form.search.elasticsearch.embedding.instructionEnabled !== original.search.elasticsearch.embedding.instructionEnabled) {
          embPatch.instructionEnabled = form.search.elasticsearch.embedding.instructionEnabled;
        }
        if (Object.keys(embPatch,).length > 0) esPatch.embedding = embPatch;
      }
      if (Object.keys(esPatch,).length > 0) searchPatch.elasticsearch = esPatch;

      const queuePatch: Record<string, unknown> = {};
      if (form.queue.backend !== original.queue.backend) queuePatch.backend = form.queue.backend;
      const kafkaPatch: Record<string, unknown> = {};
      if (JSON.stringify(form.queue.kafka.brokers,) !== JSON.stringify(original.queue.kafka.brokers,)) {
        kafkaPatch.brokers = form.queue.kafka.brokers.map((s,) => s.trim(),).filter(Boolean,);
      }
      if (Object.keys(kafkaPatch,).length > 0) queuePatch.kafka = kafkaPatch;

      const storagePatch: Record<string, unknown> = {};
      if (Object.keys(pgPatch,).length > 0) storagePatch.postgres = pgPatch;
      if (Object.keys(searchPatch,).length > 0) storagePatch.search = searchPatch;
      if (Object.keys(queuePatch,).length > 0) storagePatch.queue = queuePatch;

      const result = await updateConfig({ input: { storage: storagePatch as StorageConfigInput, }, },);
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
        <CardTitle className="font-mono text-sm text-foreground uppercase tracking-wider">{t("dashboard.storageConfig",)}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.postgresGroup",)}</legend>

          <div>
            <label htmlFor="pgHost" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgHost",)}</label>
            <Input
              id="pgHost"
              type="text"
              value={active.postgres.host}
              onChange={(e,) => updatePostgres({ host: e.target.value, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgHostHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgPort" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgPort",)}</label>
            <Input
              id="pgPort"
              type="number"
              value={active.postgres.port}
              onChange={(e,) => updatePostgres({ port: parseInt(e.target.value, 10,) || 0, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgPortHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgUsername" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgUsername",)}</label>
            <Input
              id="pgUsername"
              type="text"
              value={active.postgres.username}
              onChange={(e,) => updatePostgres({ username: e.target.value, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgUserHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgPassword" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgPassword",)}</label>
            <Input
              id="pgPassword"
              type="password"
              value={active.postgres.password}
              onChange={(e,) => updatePostgres({ password: e.target.value, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgPasswordHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgDatabase" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgName",)}</label>
            <Input
              id="pgDatabase"
              type="text"
              value={active.postgres.database}
              onChange={(e,) => updatePostgres({ database: e.target.value, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgNameHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgConnectionTimeout" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgConnectionTimeout",)}</label>
            <Input
              id="pgConnectionTimeout"
              type="number"
              value={active.postgres.connectionTimeout}
              onChange={(e,) => updatePostgres({ connectionTimeout: parseInt(e.target.value, 10,) || 0, },)}
              className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgConnectionTimeoutHint",)}</span>
          </div>

          <div>
            <label htmlFor="pgMaxConnections" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgMaxConnections",)}</label>
            <Input
              id="pgMaxConnections"
              type="number"
              value={active.postgres.maxConnections}
              onChange={(e,) => updatePostgres({ maxConnections: parseInt(e.target.value, 10,) || 0, },)}
              className={`font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.maxConnections ? "border-red" : ""}`}
            />
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgMaxConnectionsHint",)}</span>
            {errors.maxConnections && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.maxConnections}</span>}
          </div>

          <div className="flex flex-col">
            <div>
              <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgSSLMode",)}</label>
              <Select value={active.postgres.sslMode} onValueChange={(v,) => v != null && updatePostgres({ sslMode: v, },)}>
                <SelectTrigger className="w-full font-mono text-sm border-border">
                  <span className="flex-1 text-left truncate">{{
                    disable: t("dashboard.pgSSLModeDisable",),
                    allow: t("dashboard.pgSSLModeAllow",),
                    prefer: t("dashboard.pgSSLModePrefer",),
                    require: t("dashboard.pgSSLModeRequire",),
                    "verify-ca": t("dashboard.pgSSLModeVerifyCa",),
                    "verify-full": t("dashboard.pgSSLModeVerifyFull",),
                  }[active.postgres.sslMode] ?? active.postgres.sslMode}</span>
                </SelectTrigger>
                <SelectContent className="font-mono text-sm">
                  <SelectItem value="disable">{t("dashboard.pgSSLModeDisable",)}</SelectItem>
                  <SelectItem value="allow">{t("dashboard.pgSSLModeAllow",)}</SelectItem>
                  <SelectItem value="prefer">{t("dashboard.pgSSLModePrefer",)}</SelectItem>
                  <SelectItem value="require">{t("dashboard.pgSSLModeRequire",)}</SelectItem>
                  <SelectItem value="verify-ca">{t("dashboard.pgSSLModeVerifyCa",)}</SelectItem>
                  <SelectItem value="verify-full">{t("dashboard.pgSSLModeVerifyFull",)}</SelectItem>
                </SelectContent>
              </Select>
              <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgSSLModeHint",)}</span>
            </div>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.postgres.sslMode !== "disable"
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col gap-4 px-0.5">
                <div className="mt-4">
                  <label htmlFor="pgSSLCertPath" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgSSLCertPath",)}</label>
                  <Input
                    id="pgSSLCertPath"
                    type="text"
                    value={active.postgres.sslCertPath}
                    onChange={(e,) => updatePostgres({ sslCertPath: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgSSLCertPathHint",)}</span>
                </div>

                <div>
                  <label htmlFor="pgSSLKeyPath" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgSSLKeyPath",)}</label>
                  <Input
                    id="pgSSLKeyPath"
                    type="text"
                    value={active.postgres.sslKeyPath}
                    onChange={(e,) => updatePostgres({ sslKeyPath: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgSSLKeyPathHint",)}</span>
                </div>

                <div>
                  <label htmlFor="pgSSLRootCertPath" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.pgSSLRootCertPath",)}</label>
                  <Input
                    id="pgSSLRootCertPath"
                    type="text"
                    value={active.postgres.sslRootCertPath}
                    onChange={(e,) => updatePostgres({ sslRootCertPath: e.target.value, },)}
                    className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.pgSSLRootCertPathHint",)}</span>
                </div>
              </div>
            </div>
          </div>

        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.queueGroup",)}</legend>

          <div>
            <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.queueBackend",)}</label>
            <Select value={active.queue.backend} onValueChange={(v,) => v != null && updateQueue({ backend: v, },)}>
              <SelectTrigger className="w-full font-mono text-sm border-border">
                <span className="flex-1 text-left truncate">{{
                  memory: t("dashboard.queueBackendMemory",),
                  kafka: t("dashboard.queueBackendKafka",),
                }[active.queue.backend] ?? active.queue.backend}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
                <SelectItem value="memory">{t("dashboard.queueBackendMemory",)}</SelectItem>
                <SelectItem value="kafka">{t("dashboard.queueBackendKafka",)}</SelectItem>
              </SelectContent>
            </Select>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.queueBackendHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.queue.backend === "kafka"
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col px-0.5">
                <div className="mt-4">
                  <label htmlFor="kafkaBrokers" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.kafkaBrokers",)}</label>
                  <textarea
                    id="kafkaBrokers"
                    rows={4}
                    value={(active.queue.kafka.brokers ?? []).join("\n",)}
                    onChange={(e,) => {
                      const lines = e.target.value.split("\n",).filter(Boolean,);
                      if (e.target.value.endsWith("\n",)) {
                        lines.push("",);
                      }
                      updateKafka({ brokers: lines, },);
                    }}
                    className={`font-mono text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y ${errors.brokers ? "border-red" : ""}`}
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.kafkaBrokersHint",)}</span>
                  {errors.brokers && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.brokers}</span>}
                </div>
              </div>
            </div>
          </div>
        </fieldset>

        <fieldset className="border border-border rounded p-4 flex flex-col gap-4">
          <legend className="font-mono text-xs text-foreground uppercase tracking-wider px-1">{t("dashboard.searchGroup",)}</legend>

          <div>
            <label className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.searchBackend",)}</label>
            <Select value={active.search.backend} onValueChange={(v,) => v != null && updateSearch({ backend: v, },)}>
              <SelectTrigger className="w-full font-mono text-sm border-border">
                <span className="flex-1 text-left truncate">{{
                  postgresql: t("dashboard.searchBackendPg",),
                  elasticsearch: t("dashboard.searchBackendEs",),
                }[active.search.backend]}</span>
              </SelectTrigger>
              <SelectContent className="font-mono text-sm">
                <SelectItem value="postgresql">{t("dashboard.searchBackendPg",)}</SelectItem>
                <SelectItem value="elasticsearch">{t("dashboard.searchBackendEs",)}</SelectItem>
              </SelectContent>
            </Select>
            <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.searchBackendHint",)}</span>

            <div
              className={`grid transition-[grid-template-rows] duration-300 ease-in-out ${
                active.search.backend === "elasticsearch"
                  ? "grid-rows-[1fr]"
                  : "grid-rows-[0fr]"
              }`}
            >
              <div className="overflow-hidden flex flex-col px-0.5">
                <div className="mt-4">
                  <label htmlFor="esAddresses" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.esAddresses",)}</label>
                  <textarea
                    id="esAddresses"
                    rows={4}
                    value={(active.search.elasticsearch.addresses ?? []).join("\n",)}
                    onChange={(e,) => {
                      const lines = e.target.value.split("\n",).filter(Boolean,);
                      if (e.target.value.endsWith("\n",)) {
                        lines.push("",);
                      }
                      updateElasticsearch({ addresses: lines, },);
                    }}
                    className={`font-mono text-sm bg-card border border-border rounded px-3 py-2 w-full text-foreground focus-visible:ring-2 focus-visible:ring-green/50 focus-visible:outline-none resize-y ${errors.addresses ? "border-red" : ""}`}
                  />
                  <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.esAddressesHint",)}</span>
                  {errors.addresses && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.addresses}</span>}

                  <div className="mt-4">
                    <label htmlFor="embeddingEndpoint" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.embeddingEndpoint",)}</label>
                    <Input
                      id="embeddingEndpoint"
                      type="text"
                      value={active.search.elasticsearch.embedding?.endpoint ?? ""}
                      onChange={(e,) => updateElasticsearchEmbedding({ endpoint: e.target.value, },)}
                      className={`font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.embeddingEndpoint ? "border-red" : ""}`}
                    />
                    <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embeddingEndpointHint",)}</span>
                    {errors.embeddingEndpoint && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.embeddingEndpoint}</span>}
                  </div>

                  <div className="mt-4">
                    <label htmlFor="embeddingApiKey" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.embeddingApiKey",)}</label>
                    <Input
                      id="embeddingApiKey"
                      type="text"
                      value={active.search.elasticsearch.embedding?.apikey ?? ""}
                      onChange={(e,) => updateElasticsearchEmbedding({ apikey: e.target.value, },)}
                      className="font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50"
                    />
                    <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embeddingApiKeyHint",)}</span>
                  </div>

                  <div className="mt-4 flex flex-col">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <Checkbox
                        checked={active.search.elasticsearch.embedding?.instructionEnabled ?? false}
                        onCheckedChange={(v: boolean,) => updateElasticsearchEmbedding({ instructionEnabled: v, },)}
                        className="size-4"
                      />
                      <span className="font-mono text-xs text-foreground uppercase">{t("dashboard.embeddingInstructionEnabled",)}</span>
                    </label>
                    <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embeddingInstructionEnabledHint",)}</span>
                  </div>

                  <div className="mt-4">
                    <label htmlFor="embeddingModel" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.embeddingModel",)}</label>
                    <Input
                      id="embeddingModel"
                      type="text"
                      value={active.search.elasticsearch.embedding?.model ?? ""}
                      onChange={(e,) => updateElasticsearchEmbedding({ model: e.target.value, },)}
                      className={`font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.embeddingModel ? "border-red" : ""}`}
                    />
                    <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embeddingModelHint",)}</span>
                    {errors.embeddingModel && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.embeddingModel}</span>}
                  </div>

                  <div className="mt-4">
                    <label htmlFor="embeddingDimensions" className="font-mono text-xs text-foreground uppercase block mb-1.5">{t("dashboard.embeddingDimensions",)}</label>
                    <Input
                      id="embeddingDimensions"
                      type="number"
                      min={1}
                      value={active.search.elasticsearch.embedding?.dimensions ?? 1024}
                      onChange={(e,) => updateElasticsearchEmbedding({ dimensions: parseInt(e.target.value, 10,) || 0, },)}
                      className={`font-mono text-sm bg-card border-border focus-visible:ring-2 focus-visible:ring-green/50 ${errors.embeddingDimensions ? "border-red" : ""}`}
                    />
                    <span className="font-mono text-[10px] text-muted-foreground/60 mt-1 block">{t("dashboard.embeddingDimensionsHint",)}</span>
                    {errors.embeddingDimensions && <span className="font-mono text-[10px] text-red mt-0.5 block">{errors.embeddingDimensions}</span>}
                  </div>

                  <div className="mt-4">
                    <Button
                      variant="default"
                      size="sm"
                      className="gap-1.5 font-mono text-xs bg-green/20 text-green border border-green/40 hover:bg-green/30 disabled:opacity-50"
                      onClick={startReindex}
                      disabled={reindexing}
                    >
                      {reindexing ? (
                        <>
                          <LoaderCircle className="size-3.5 animate-spin" />
                          {t("dashboard.reindexing", { indexed: reindexIndexed, total: reindexTotal, },)}
                        </>
                      ) : (
                        t("dashboard.reindexToEs",)
                      )}
                    </Button>
                  </div>
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
