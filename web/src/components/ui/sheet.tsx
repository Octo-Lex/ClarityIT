import { Drawer as DrawerPrimitive } from "@base-ui/react/drawer"
import { cva, type VariantProps } from "class-variance-authority"
import * as React from "react"

import { cn } from "@/lib/utils"

const SheetSide = ["top", "right", "bottom", "left"] as const
type SheetSide = (typeof SheetSide)[number]

const sideToDrawerSide: Record<SheetSide, "top" | "right" | "bottom" | "left"> = {
  top: "top",
  right: "right",
  bottom: "bottom",
  left: "left",
}

function Sheet({
  ...props
}: React.ComponentProps<typeof DrawerPrimitive.Root>) {
  return <DrawerPrimitive.Root {...props} />
}

function SheetTrigger({
  ...props
}: React.ComponentProps<typeof DrawerPrimitive.Trigger>) {
  return <DrawerPrimitive.Trigger {...props} />
}

function SheetClose({
  ...props
}: React.ComponentProps<typeof DrawerPrimitive.Close>) {
  return <DrawerPrimitive.Close {...props} />
}

function SheetContent({
  className,
  children,
  side = "right",
  title,
  description,
  ...props
}: Omit<React.ComponentProps<typeof DrawerPrimitive.Content>, "title"> & {
  side?: SheetSide
  title?: React.ReactNode
  description?: React.ReactNode
}) {
  return (
    <DrawerPrimitive.Portal>
      <DrawerPrimitive.Backdrop className="fixed inset-0 z-50 bg-black/50 data-[starting-style]:animate-in data-[starting-style]:fade-in-0 data-[ending-style]:animate-out data-[ending-style]:fade-out-0" />
      <DrawerPrimitive.Popup
        data-slot="sheet-content"
        data-side={side}
        className={cn(
          "fixed z-50 flex flex-col gap-4 bg-background shadow-lg transition ease-in-out data-[starting-style]:animate-in data-[ending-style]:animate-out",
          side === "top" &&
            "inset-x-0 top-0 h-auto border-b data-[starting-style]:slide-in-from-top data-[ending-style]:slide-out-to-top",
          side === "bottom" &&
            "inset-x-0 bottom-0 h-auto border-t data-[starting-style]:slide-in-from-bottom data-[ending-style]:slide-out-to-bottom",
          side === "left" &&
            "inset-y-0 left-0 h-full w-3/4 max-w-sm border-r data-[starting-style]:slide-in-from-left data-[ending-style]:slide-out-to-left",
          side === "right" &&
            "inset-y-0 right-0 h-full w-3/4 max-w-sm border-l data-[starting-style]:slide-in-from-right data-[ending-style]:slide-out-to-right",
          className,
        )}
        title={undefined}
        {...props}
      >
        {(title || description) && (
          <div className="flex flex-col gap-1 border-b px-4 py-3">
            {title && (
              <DrawerPrimitive.Title className="font-heading text-base font-medium leading-none">
                {title}
              </DrawerPrimitive.Title>
            )}
            {description && (
              <DrawerPrimitive.Description className="text-sm text-muted-foreground">
                {description}
              </DrawerPrimitive.Description>
            )}
          </div>
        )}
        <div className="flex-1 overflow-auto px-4 pb-4">{children}</div>
      </DrawerPrimitive.Popup>
    </DrawerPrimitive.Portal>
  )
}

function SheetHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-header"
      className={cn("flex flex-col gap-1 text-center sm:text-left", className)}
      {...props}
    />
  )
}

function SheetFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="sheet-footer"
      className={cn(
        "mt-auto flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
        className,
      )}
      {...props}
    />
  )
}

function SheetTitle({
  className,
  ...props
}: React.ComponentProps<typeof DrawerPrimitive.Title>) {
  return (
    <DrawerPrimitive.Title
      data-slot="sheet-title"
      className={cn("font-heading text-base font-medium text-foreground", className)}
      {...props}
    />
  )
}

function SheetDescription({
  className,
  ...props
}: React.ComponentProps<typeof DrawerPrimitive.Description>) {
  return (
    <DrawerPrimitive.Description
      data-slot="sheet-description"
      className={cn("text-sm text-muted-foreground", className)}
      {...props}
    />
  )
}

export {
  Sheet,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
  SheetSide,
}
