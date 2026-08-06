import { Checkbox as CheckboxPrimitive, type CheckboxRootProps, } from "@base-ui/react/checkbox";
import { cn, } from "@/lib/utils";
import { CheckIcon, MinusIcon, } from "lucide-react";

function Checkbox({ className, ...props }: CheckboxRootProps,) {
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      className={cn(
        "group/checkbox inline-flex size-4 shrink-0 items-center justify-center rounded border border-input dark:border-foreground/20 bg-background transition-all outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-50 data-[checked]:border-primary data-[checked]:bg-primary data-[checked]:text-primary-foreground",
        className,
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator className="flex items-center justify-center data-[checked]:animate-in data-[checked]:fade-in-0 data-[checked]:zoom-in-90">
        {props.indeterminate ? <MinusIcon className="size-3" /> : <CheckIcon className="size-3" />}
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

export { Checkbox, };
