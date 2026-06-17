import { cva, type VariantProps } from "class-variance-authority"
import * as React from "react"

import { cn } from "@/lib/utils"

/**
 * Unified status-tone badge. Replaces the scattered legacy `.badge-green`,
 * `.badge-yellow`, `.badge-red`, `.badge-blue`, `.badge-gray` classes with a
 * single semantic component. Used for incident severity, run state, risk level,
 * approval status, etc.
 *
 * Tones map to design tokens:
 *  - success: ok, resolved, active, healthy
 *  - warning: pending, sev3, medium risk
 *  - danger:  failed, blocked, sev1/sev2, critical
 *  - info:    running, in-progress
 *  - neutral: draft, disabled, default
 */
const statusBadgeVariants = cva(
  "inline-flex h-5 w-fit shrink-0 items-center justify-center gap-1 rounded-full border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-colors",
  {
    variants: {
      tone: {
        success: "bg-success/15 text-success",
        warning: "bg-warning/20 text-warning",
        danger: "bg-destructive/15 text-destructive",
        info: "bg-info/15 text-info",
        neutral: "bg-muted text-muted-foreground",
      },
      dot: {
        true: "pl-1.5",
        false: "",
      },
    },
    defaultVariants: {
      tone: "neutral",
      dot: false,
    },
  },
)

function StatusBadge({
  className,
  tone,
  dot = false,
  children,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof statusBadgeVariants>) {
  return (
    <span
      data-slot="status-badge"
      data-tone={tone}
      className={cn(statusBadgeVariants({ tone, dot }), className)}
      {...props}
    >
      {dot && (
        <span
          aria-hidden="true"
          className={cn(
            "size-1.5 rounded-full",
            tone === "success" && "bg-success",
            tone === "warning" && "bg-warning",
            tone === "danger" && "bg-destructive",
            tone === "info" && "bg-info",
            (!tone || tone === "neutral") && "bg-muted-foreground",
          )}
        />
      )}
      {children}
    </span>
  )
}

export { StatusBadge, statusBadgeVariants }
