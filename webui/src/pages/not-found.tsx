import { useTranslation, } from "react-i18next";
import { Button, } from "@/components/ui/button";
import { Link, } from "react-router";
import { useDocumentTitle, } from "@/lib/use-document-title";

export function NotFoundPage() {
  const { t, } = useTranslation();
  useDocumentTitle(t("title.notFound",),);
  return (
    <main className="mx-auto max-w-screen-xl px-4 py-12 sm:py-24">
      <div className="flex flex-col items-center gap-6 text-center">
        <h1 className="font-mono text-2xl font-bold text-muted-foreground sm:text-4xl">
          <span className="text-amber">#</span> {t("error.notFoundTitle",)}
        </h1>
        <p className="font-mono text-sm text-muted-foreground max-w-md">{t("error.notFoundMessage",)}</p>
        <Link to="/browse">
          <Button variant="outline" size="sm">
            {t("common.backToBrowse",)}
          </Button>
        </Link>
      </div>
    </main>
  );
}
