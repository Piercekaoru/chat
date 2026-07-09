"use client";

import * as React from "react";
import { useTranslations } from "next-intl";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { LoginPanel } from "@/features/auth/components/login-page";

type LoginDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Where the standalone login page would navigate after auth. In dialog mode we do not
   * navigate; we pass this through so `useLoginPage` keeps a consistent next-path for
   * provider (OAuth) redirects, which do leave the page.
   */
  nextPath?: string;
  /**
   * Invoked after a successful in-place login/registration. The session snapshot has
   * already been written and broadcast by then, so callers typically just close the dialog.
   */
  onAuthenticated?: () => void;
};

export function LoginDialog({ open, onOpenChange, nextPath = "", onAuthenticated }: LoginDialogProps) {
  const t = useTranslations("login");

  const handleAuthenticated = React.useCallback(() => {
    onAuthenticated?.();
    onOpenChange(false);
  }, [onAuthenticated, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader className="sr-only">
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("title")}</DialogDescription>
        </DialogHeader>
        <div className="flex justify-center py-2">
          <LoginPanel nextPath={nextPath} onAuthenticated={handleAuthenticated} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
