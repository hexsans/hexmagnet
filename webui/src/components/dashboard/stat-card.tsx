import { Card, CardContent, } from "@/components/ui/card";
import { cn, } from "@/lib/utils";
import type { LucideIcon, } from "lucide-react";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  icon: LucideIcon;
  accent?: "green" | "cyan" | "amber" | "destructive";
}

const ACCENT_MAP = {
  green: {
    icon: "text-green",
    value: "text-green",
    border: "border-green/20",
    bg: "bg-green/5",
  },
  cyan: {
    icon: "text-cyan",
    value: "text-cyan",
    border: "border-cyan/20",
    bg: "bg-cyan/5",
  },
  amber: {
    icon: "text-amber",
    value: "text-amber",
    border: "border-amber/20",
    bg: "bg-amber/5",
  },
  destructive: {
    icon: "text-destructive",
    value: "text-destructive",
    border: "border-destructive/20",
    bg: "bg-destructive/5",
  },
};

export function StatCard({ label, value, sub, icon: Icon, accent = "green", }: StatCardProps,) {
  const colors = ACCENT_MAP[accent];
  return (
    <Card className={cn("border", colors.border, colors.bg,)}>
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <p className="font-mono text-xs text-muted-foreground uppercase tracking-wider">{label}</p>
            <p className={cn("font-mono text-2xl font-bold mt-1", colors.value,)}>{value}</p>
            {sub && <p className="font-mono text-xs text-muted-foreground mt-0.5">{sub}</p>}
          </div>
          <div className={cn("rounded-md border p-2", colors.border, colors.bg,)}>
            <Icon className={cn("size-5", colors.icon,)} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
