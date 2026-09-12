import { useState, } from "react";
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

  const hide = (target: ResumeJob,) => {
    setHidden((prev,) => [...prev, target,],);
  };

  const handleContinue = () => {
    setBusy(true,);

    if (job === "reindex") {
      reindexMutation({},).then((result,) => {
        setBusy(false,);
        hide("reindex",);
        refresh();

        const error = result.data?.torrent?.reindexToElasticsearch?.error;
        if (error) toast.error(error,);
      },);

      return;
    }

    reclassifyMutation({},).then((result,) => {
      setBusy(false,);
      hide("reclassify",);
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
