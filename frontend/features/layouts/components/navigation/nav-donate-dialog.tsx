"use client"

import * as React from "react"
import { QRCodeSVG } from "qrcode.react"
import { Check, Copy } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useCopyAction } from "@/shared/components/copy-action"
import {
  DONATE_NETWORK_LABEL,
  DONATE_SOLANA_ADDRESS,
  truncateAddress,
} from "@/features/layouts/model/donate"

export function NavDonateDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const t = useTranslations("common.navigation")
  const { copy, isCopied } = useCopyAction({
    messages: {
      copied: t("donateCopied"),
      failed: t("donateCopyFailed"),
    },
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[400px]">
        <DialogHeader className="items-center text-center">
          <DialogTitle className="text-base">{t("donateTitle")}</DialogTitle>
          <DialogDescription>{t("donateDescription")}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col items-center gap-4 py-2">
          <div className="rounded-xl border border-border/60 bg-white p-3">
            <QRCodeSVG
              value={DONATE_SOLANA_ADDRESS}
              size={168}
              level="M"
              marginSize={0}
            />
          </div>

          <div className="flex w-full flex-col items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">
              {DONATE_NETWORK_LABEL}
            </span>
            <div className="flex w-full items-center gap-2 rounded-md border border-border/60 bg-muted/40 px-3 py-2">
              <code
                className="min-w-0 flex-1 truncate font-mono text-xs"
                title={DONATE_SOLANA_ADDRESS}
              >
                {truncateAddress(DONATE_SOLANA_ADDRESS, 6, 6)}
              </code>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="shrink-0"
                aria-label={t("donateCopyAria")}
                onClick={() => void copy(DONATE_SOLANA_ADDRESS)}
              >
                {isCopied() ? (
                  <Check className="size-3.5" />
                ) : (
                  <Copy className="size-3.5" />
                )}
              </Button>
            </div>
          </div>

          <p className="text-center text-xs text-muted-foreground">
            {t("donateThanks")}
          </p>
        </div>
      </DialogContent>
    </Dialog>
  )
}
