import { useTranslation, } from "react-i18next";
import { Button, } from "@/components/ui/button";
import { useDocumentTitle, } from "@/lib/use-document-title";

export function ErrorPage({ error, reset, }: { error: Error & { digest?: string }; reset: () => void },) {
  const { t, } = useTranslation();
  useDocumentTitle(t("title.error",),);
  return (
    <main className="mx-auto max-w-screen-xl px-4 py-12 sm:py-24">
      <div className="flex flex-col items-center gap-6 text-center">
        <h1 className="font-mono text-2xl font-bold text-muted-foreground sm:text-4xl">
          <span className="text-destructive">!</span> {t("error.title",)}
        </h1>
        <p className="font-mono text-sm text-muted-foreground max-w-md">{t("error.message",)}</p>
        {error.digest && <p className="font-mono text-xs text-muted-foreground/60">{t("error.digest", { digest: error.digest, },)}</p>}
        {error.message && <pre className="max-w-md overflow-x-auto font-mono text-xs text-destructive whitespace-pre-wrap break-words">{error.message}</pre>}
        <Button variant="outline" size="sm" onClick={reset}>
          {t("common.tryAgain",)}
        </Button>
      </div>
    </main>
  );
}
