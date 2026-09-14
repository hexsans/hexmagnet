import { useEffect, useRef, useState, } from "react";
import { useTranslation, } from "react-i18next";
import { useMutation, } from "urql";
import { LoaderCircle, } from "lucide-react";
import { toast, } from "sonner";
import { Button, } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, } from "@/components/ui/dialog";
import { useMaintenanceStatus, } from "@/lib/use-maintenance-status";
import {
  DiscardReclassifyProgressDocument,
  DiscardReindexProgressDocument,
  ReclassifyTorrentsDocument,
  ReindexElasticsearchDocument,
} from "@/lib/graphql/generated/graphql";

type ResumeJob = "reindex" | "reclassify";

export function MaintenanceResumeDialog() {
  const { t, } = useTranslation();
  const { reindex, reclassify, refresh, } = useMaintenanceStatus();

  const [, reindexMutation,] = useMutation(ReindexElasticsearchDocument,);
  const [, reclassifyMutation,] = useMutation(ReclassifyTorrentsDocument,);
  const [, discardReindexMutation,] = useMutation(DiscardReindexProgressDocument,);
  const [, discardReclassifyMutation,] = useMutation(DiscardReclassifyProgressDocument,);

  const [busy, setBusy,] = useState(false,);
  const [hidden, setHidden,] = useState<ResumeJob[]>([],);
  const previousRunning = useRef<Record<ResumeJob, boolean>>({ reindex: false, reclassify: false, },);

  // Re-show the dialog when a run that was started from it stops resumable
  // again (for example an LLM/embedding failure immediately after Continue).
  useEffect(() => {
    const running: Record<ResumeJob, boolean> = {
      reindex: Boolean(reindex?.running,),
      reclassify: Boolean(reclassify?.running,),
    };
    const resumable: Record<ResumeJob, boolean> = {
      reindex: Boolean(reindex?.resumable,),
      reclassify: Boolean(reclassify?.resumable,),
    };

    for (const target of ["reindex", "reclassify",] as ResumeJob[]) {
      if (previousRunning.current[target] && !running[target] && resumable[target]) {
        setHidden((prev,) => prev.filter((item,) => item !== target,),);
      }
    }

    previousRunning.current = running;
  }, [reindex, reclassify,],);

  const anyRunning = Boolean(reindex?.running,) || Boolean(reclassify?.running,);

  const pending: ResumeJob | null = anyRunning
    ? null
    : reindex?.resumable
      ? "reindex"
      : reclassify?.resumable
        ? "reclassify"
        : null;

  const job = pending && !hidden.includes(pending,) ? pending : null;

  if (!job) return null;

  const configChanged = job === "reindex" ? Boolean(reindex?.configChanged,) : Boolean(reclassify?.configChanged,);
  const error = job === "reindex" ? reindex?.error : reclassify?.error;

  const hide = (target: ResumeJob,) => {
    setHidden((prev,) => [...prev, target,],);
  };

  const handleContinue = () => {
    setBusy(true,);

    // The dialog intentionally stays mounted after Continue: it disappears
    // while the status reports running, and reappears automatically if the
    // run pauses again (e.g. the LLM or embedding endpoint is still down).
    if (job === "reindex") {
      reindexMutation({},).then((result,) => {
        setBusy(false,);
        refresh();

        const error = result.data?.torrent?.reindexToElasticsearch?.error;
        if (error) toast.error(error,);
      },);

      return;
    }

    reclassifyMutation({},).then((result,) => {
      setBusy(false,);
      refresh();

      const error = result.data?.torrent?.reclassifyTorrents?.error;
      if (error) toast.error(error,);
    },);
  };

  const handleCancel = () => {
    setBusy(true,);

    if (job === "reindex") {
      discardReindexMutation({},).then(() => {
        setBusy(false,);
        hide("reindex",);
        refresh();
      },);

      return;
    }

    discardReclassifyMutation({},).then(() => {
      setBusy(false,);
      hide("reclassify",);
      refresh();
    },);
  };

  return (
    <Dialog open onOpenChange={(open,) => { if (!open) hide(job,); }}>
      <DialogContent className="font-mono">
        <DialogHeader>
          <DialogTitle>
            {job === "reindex" ? t("dashboard.resumeReindexTitle",) : t("dashboard.resumeReclassifyTitle",)}
          </DialogTitle>
          <DialogDescription className="text-xs">
            {job === "reindex"
              ? t("dashboard.resumeReindexProgress", { indexed: reindex?.indexed ?? 0, total: reindex?.total ?? 0, },)
              : t("dashboard.resumeReclassifyProgress", { processed: reclassify?.processed ?? 0, total: reclassify?.total ?? 0, },)}
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="rounded-lg border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            <div className="font-medium">{t("dashboard.resumePausedByError",)}</div>
            <div className="mt-1 break-all whitespace-pre-wrap">{error}</div>
          </div>
        )}

        {configChanged && (
          <div className="rounded-lg border border-amber/40 bg-amber/10 px-3 py-2 text-xs text-amber">
            {job === "reindex" ? t("dashboard.resumeReindexConfigChanged",) : t("dashboard.resumeReclassifyConfigChanged",)}
          </div>
        )}

        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" className="font-mono text-xs" disabled={busy} onClick={handleCancel}>
            {t("dashboard.resumeCancel",)}
          </Button>
          <Button variant="default" size="sm" className="font-mono text-xs" disabled={busy} onClick={handleContinue}>
            {busy && <LoaderCircle className="size-3.5 animate-spin" />}
            {t("dashboard.resumeContinue",)}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
