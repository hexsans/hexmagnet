import { useState, } from "react";
import { useMutation, } from "urql";
import { useTranslation, } from "react-i18next";
import { Copy, CopyCheck, RefreshCw, X, } from "lucide-react";
import { toast, } from "sonner";
import { Button, } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger, } from "@/components/ui/tooltip";
import { TorrentReprocessDocument, } from "@/lib/graphql/generated/graphql";
import type { Torrent, } from "@/lib/types";

interface BulkActionsToolbarProps {
  selectedTorrents: Torrent[];
  onMutate: () => void;
  onClearSelection: () => void;
}

export function BulkActionsToolbar({ selectedTorrents, onMutate, onClearSelection, }: BulkActionsToolbarProps,) {
  const { t, } = useTranslation();
  const [copiedMagnets, setCopiedMagnets,] = useState(false,);
  const [copiedHashes, setCopiedHashes,] = useState(false,);
  const [, reprocessTorrents,] = useMutation(TorrentReprocessDocument,);

  const infoHashes = selectedTorrents.map((t,) => t.infoHash,);

  function copyMagnets() {
    const text = selectedTorrents.map((t,) => t.magnetLink,).join("\n",);
    navigator.clipboard.writeText(text,);
    setCopiedMagnets(true,);
    toast.success(t("toast.magnetCopiedMulti", { count: selectedTorrents.length, },),);
    setTimeout(() => setCopiedMagnets(false,), 2000,);
  }

  function copyHashes() {
    const text = selectedTorrents.map((t,) => t.infoHash,).join("\n",);
    navigator.clipboard.writeText(text,);
    setCopiedHashes(true,);
    toast.success(t("toast.infoHashCopiedMulti", { count: selectedTorrents.length, },),);
    setTimeout(() => setCopiedHashes(false,), 2000,);
  }

  async function handleReprocess() {
    const result = await reprocessTorrents({
      input: { infoHashes, },
    },);
    if (result.error) {
      toast.error(t("toast.failed.reprocess", { message: result.error.message, },),);
      return;
    }
    toast.success(t("toast.reprocessing", { count: infoHashes.length, },),);
    onClearSelection();
    onMutate();
  }

  return (
    <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
      <span className="font-mono text-xs text-muted-foreground shrink-0">{t("common.selected", { count: selectedTorrents.length, },)}</span>
      <div className="flex items-center gap-1">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="sm"
                className="gap-1 font-mono text-xs text-muted-foreground hover:text-green"
                onClick={copyMagnets}
              />
            }
          >
            {copiedMagnets ? <CopyCheck className="size-3.5" /> : <Copy className="size-3.5" />}
            {t("common.magnets",)}
          </TooltipTrigger>
          <TooltipContent>{t("tooltip.copyMagnets",)}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="sm"
                className="gap-1 font-mono text-xs text-muted-foreground hover:text-cyan"
                onClick={copyHashes}
              />
            }
          >
            {copiedHashes ? <CopyCheck className="size-3.5" /> : <Copy className="size-3.5" />}
            {t("common.infoHashes",)}
          </TooltipTrigger>
          <TooltipContent>{t("tooltip.copyInfoHashes",)}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="sm"
                className="gap-1 font-mono text-xs text-muted-foreground hover:text-cyan"
                onClick={handleReprocess}
              />
            }
          >
            <RefreshCw className="size-3.5" />
            {t("detail.reprocess",)}
          </TooltipTrigger>
          <TooltipContent>{t("tooltip.reprocessTooltip",)}</TooltipContent>
        </Tooltip>
      </div>

      <div className="ml-auto">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-muted-foreground hover:text-destructive"
                onClick={onClearSelection}
              />
            }
          >
            <X className="size-3.5" />
            <span className="sr-only">{t("tooltip.clearSelection",)}</span>
          </TooltipTrigger>
          <TooltipContent>{t("tooltip.clearSelection",)}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
