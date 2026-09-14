import { useState, useRef, useEffect, } from "react";
import { useTranslation, } from "react-i18next";

interface PageInputProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number,) => void;
}

export function PageInput({ page, totalPages, onPageChange, }: PageInputProps,) {
  const { t, } = useTranslation();
  const [editing, setEditing,] = useState(false,);
  const [editValue, setEditValue,] = useState(String(page,),);
  const inputRef = useRef<HTMLInputElement>(null,);

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editing,],);

  function submit() {
    const n = parseInt(editValue, 10,);
    if (!isNaN(n,) && n >= 1 && n <= totalPages) onPageChange(n,);
    setEditing(false,);
  }

  function onKeyDown(e: React.KeyboardEvent,) {
    if (e.key === "Enter") submit();
    else if (e.key === "Escape") {
      setEditValue(String(page,),);
      setEditing(false,);
    }
  }

  if (editing) {
    return (
      <span className="inline-flex items-center gap-1">
        <input
          ref={inputRef}
          type="text"
          inputMode="numeric"
          value={editValue}
          onChange={(e,) => setEditValue(e.target.value,)}
          onBlur={submit}
          onKeyDown={onKeyDown}
          className="inline h-8 w-14 bg-card border border-border rounded text-center font-mono text-base text-foreground outline-none focus:border-green/50 pointer-coarse:h-10 md:h-5 md:w-8 md:text-xs"
        />
        <span className="text-muted-foreground">/ {totalPages}</span>
      </span>
    );
  }

  return (
    <span
      className="text-foreground cursor-pointer hover:text-green transition-colors"
      onClick={() => {
        setEditValue(String(page,),);
        setEditing(true,);
      }}
      title={t("common.jumpToPage",)}
    >
      {page} <span className="text-muted-foreground">/ {totalPages}</span>
    </span>
  );
}
