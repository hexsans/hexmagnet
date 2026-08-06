import { useState, useRef, useEffect, } from "react";
import { useTranslation, } from "react-i18next";
import { Languages, } from "lucide-react";
import { Button, } from "@/components/ui/button";
import { LANG_NORMALIZE, } from "@/lib/constants";

const LANGUAGES = [
  { code: "en", label: "English", },
  { code: "zh-Hans", label: "中文", },
  { code: "ja", label: "日本語", },
  { code: "ko", label: "한국어", },
  { code: "fr", label: "Français", },
  { code: "it", label: "Italiano", },
  { code: "ru", label: "Русский", },
  { code: "nl", label: "Nederlands", },
];

export function LanguageSwitcher() {
  const { i18n, t, } = useTranslation();
  const [open, setOpen,] = useState(false,);
  const ref = useRef<HTMLDivElement>(null,);
  const currentLang = LANG_NORMALIZE[i18n.language] ?? i18n.language;

  useEffect(() => {
    function handleClickOutside(e: MouseEvent,) {
      if (ref.current && !ref.current.contains(e.target as Node,)) {
        setOpen(false,);
      }
    }
    document.addEventListener("mousedown", handleClickOutside,);
    return () => document.removeEventListener("mousedown", handleClickOutside,);
  }, [],);

  return (
    <div ref={ref} className="relative">
      <Button
        variant="ghost"
        size="icon"
        className="size-8 text-muted-foreground hover:text-foreground"
        onClick={() => setOpen(!open,)}
        aria-label={t("common.switchLanguage",)}
      >
        <Languages className="size-4" />
      </Button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-50 min-w-24 rounded-lg bg-popover text-popover-foreground shadow-md ring-1 ring-foreground/10 overflow-hidden">
          {LANGUAGES.map(({ code, label, },) => (
            <button
              key={code}
              className={`block w-full px-3 py-1.5 text-left text-sm font-mono transition-colors hover:bg-accent hover:text-accent-foreground ${
                currentLang === code ? "bg-accent text-accent-foreground" : ""
              }`}
              onClick={() => {
                i18n.changeLanguage(code,);
                setOpen(false,);
              }}
            >
              {label}
            </button>
          ),)}
        </div>
      )}
    </div>
  );
}
