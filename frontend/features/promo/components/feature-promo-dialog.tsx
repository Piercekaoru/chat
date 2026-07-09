"use client";

import * as React from "react";
import { usePathname } from "next/navigation";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { readLocalStorageItem, writeLocalStorageItem } from "@/shared/lib/storage-key-migration";

// Bump the version suffix whenever a new feature promo should be shown again.
const PROMO_STORAGE_KEY = "promo:grok-4-5:v1";
const PROMO_SEEN_VALUE = "1";

function isSkippedPath(pathname: string | null): boolean {
  if (!pathname) {
    return false;
  }
  return pathname === "/share" || pathname.startsWith("/share/");
}

export function FeaturePromoDialog() {
  const t = useTranslations("promo");
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = React.useState(false);

  React.useEffect(() => {
    if (isSkippedPath(pathname)) {
      return;
    }
    const seen = readLocalStorageItem(PROMO_STORAGE_KEY);
    if (seen !== PROMO_SEEN_VALUE) {
      setOpen(true);
    }
  }, [pathname]);

  const dismiss = React.useCallback(() => {
    writeLocalStorageItem(PROMO_STORAGE_KEY, PROMO_SEEN_VALUE);
    setOpen(false);
  }, []);

  const start = React.useCallback(() => {
    dismiss();
    router.push("/chat");
  }, [dismiss, router]);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          dismiss();
        }
      }}
    >
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-[440px]">
        <div className="relative h-48 w-full overflow-hidden">
          <div className="absolute inset-0 bg-gradient-to-br from-emerald-200 via-teal-100 to-sky-200 dark:from-emerald-900/40 dark:via-teal-900/30 dark:to-sky-900/40" />
          <div className="absolute -left-10 -top-12 size-44 rounded-full bg-lime-300/70 blur-3xl dark:bg-lime-500/30" />
          <div className="absolute -right-8 top-4 size-40 rounded-full bg-sky-300/70 blur-3xl dark:bg-sky-500/30" />
          <div className="absolute -bottom-14 left-1/3 size-44 rounded-full bg-emerald-300/60 blur-3xl dark:bg-emerald-500/30" />

          <div className="absolute inset-0 flex items-center justify-center px-8">
            <div className="rounded-[1.75rem] bg-white/95 px-8 py-5 shadow-xl ring-1 ring-black/5 backdrop-blur-sm">
              <span className="whitespace-nowrap text-3xl font-semibold tracking-tight text-neutral-900">
                Grok <span className="text-emerald-600">4.5</span>
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-4 px-6 pb-6 pt-5">
          <DialogHeader>
            <DialogTitle className="text-base">{t("title")}</DialogTitle>
            <DialogDescription>{t("description")}</DialogDescription>
          </DialogHeader>
          <DialogFooter className="[&_[data-slot=button][data-size=default]]:h-9">
            <Button type="button" variant="ghost" onClick={dismiss}>
              {t("dismiss")}
            </Button>
            <Button type="button" onClick={start}>
              {t("cta")}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
