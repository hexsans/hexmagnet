import { useCallback, useEffect, useState, } from "react";
import { useClient, } from "urql";
import { ReindexStatusDocument, ReclassifyStatusDocument, } from "@/lib/graphql/generated/graphql";

export interface ReindexStatus {
  total: number;
  indexed: number;
  done: boolean;
  running: boolean;
  resumable: boolean;
  configChanged: boolean;
  error: string | null;
}

export interface ReclassifyStatus {
  total: number;
  processed: number;
  done: boolean;
  running: boolean;
  resumable: boolean;
  configChanged: boolean;
  error: string | null;
}

export interface MaintenanceStatus {
  reindex: ReindexStatus | null;
  reclassify: ReclassifyStatus | null;
  reindexRunning: boolean;
  reclassifyRunning: boolean;
  anyRunning: boolean;
  refresh: () => void;
}

function isActive(status: { done: boolean; running: boolean } | null | undefined,): boolean {
  return Boolean(status && status.running && !status.done,);
}

export function useMaintenanceStatus(pollMs = 2000,): MaintenanceStatus {
  const client = useClient();
  const [reindex, setReindex,] = useState<ReindexStatus | null>(null,);
  const [reclassify, setReclassify,] = useState<ReclassifyStatus | null>(null,);
  const [tick, setTick,] = useState(0,);

  useEffect(() => {
    let cancelled = false;

    const poll = () => {
      client.query(ReindexStatusDocument, {}, { requestPolicy: "network-only", },).toPromise().then((r,) => {
        if (!cancelled && r.data?.reindexStatus) setReindex(r.data.reindexStatus,);
      },).catch(() => {},);
      client.query(ReclassifyStatusDocument, {}, { requestPolicy: "network-only", },).toPromise().then((r,) => {
        if (!cancelled && r.data?.reclassifyStatus) setReclassify(r.data.reclassifyStatus,);
      },).catch(() => {},);
    };

    poll();
    const id = setInterval(poll, pollMs,);

    return () => {
      cancelled = true;
      clearInterval(id,);
    };
  }, [client, pollMs, tick,],);

  const refresh = useCallback(() => setTick((v,) => v + 1,), [],);

  return {
    reindex,
    reclassify,
    reindexRunning: isActive(reindex,),
    reclassifyRunning: isActive(reclassify,),
    anyRunning: isActive(reindex,) || isActive(reclassify,),
    refresh,
  };
}
