"use client";

import * as React from "react";
import { Plus } from "lucide-react";
import { useTranslations } from "next-intl";

import { Send } from "@/components/animate-ui/icons/send";
import { Button } from "@/components/ui/button";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupTextarea,
} from "@/components/ui/input-group";
import { ChatEmptyState } from "@/features/chat/components/sections/chat-empty";
import { LoginDialog } from "@/features/auth/components/login-dialog";
import { useChatViewerProfile } from "@/features/chat/hooks/use-chat-viewer-profile";
import { APP_BRAND_NAME } from "@/shared/brand";

/**
 * Landing screen shown at `/chat` for guests (no session). It mirrors the look of the
 * real empty composer but is entirely read-only: any interaction opens the login dialog
 * instead of mounting the authenticated chat runtime. Logging in writes a session
 * snapshot whose broadcast makes the workspace shell swap to the authed chat in place.
 */
export function GuestChatLanding() {
  const t = useTranslations("chat.guest");
  const tComposer = useTranslations("chat.composer");
  const { greetingTitle } = useChatViewerProfile();
  const [loginOpen, setLoginOpen] = React.useState(false);

  const promptLogin = React.useCallback(() => {
    setLoginOpen(true);
  }, []);

  return (
    <div className="relative flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden">
      <div className="flex items-center justify-end gap-2 px-4 py-3 md:px-6">
        <Button
          variant="ghost"
          size="sm"
          className="rounded-md"
          onClick={promptLogin}
        >
          {t("signIn")}
        </Button>
        <Button
          size="sm"
          className="rounded-md"
          onClick={promptLogin}
        >
          {t("register")}
        </Button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        <ChatEmptyState greetingTitle={greetingTitle} showBrandLogo>
          <div className="mx-auto w-full max-w-[1080px] px-3 pb-6 md:px-0">
            <button
              type="button"
              aria-label={t("startPrompt")}
              onClick={promptLogin}
              className="w-full cursor-text text-left outline-none"
            >
              <InputGroup className="pointer-events-none rounded-3xl border-input/40">
                <InputGroupTextarea
                  tabIndex={-1}
                  readOnly
                  rows={1}
                  placeholder={tComposer("inputPlaceholder")}
                  style={{ fontFamily: "var(--font-chat)", fontWeight: "var(--font-chat-weight)" }}
                  className="min-h-12 rounded-3xl px-5 pt-4 text-[15px] leading-6"
                />
                <InputGroupAddon align="block-end" className="items-center justify-between pt-2">
                  <InputGroupButton
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="size-7 rounded-md text-muted-foreground sm:size-8"
                    aria-hidden
                  >
                    <Plus size={20} strokeWidth={1.4} />
                  </InputGroupButton>
                  <InputGroupButton
                    type="button"
                    size="icon-sm"
                    className="size-7 rounded-full sm:size-8"
                    aria-hidden
                  >
                    <Send size={18} />
                  </InputGroupButton>
                </InputGroupAddon>
              </InputGroup>
            </button>
            <p className="mt-3 text-center text-xs text-muted-foreground">
              {t("hint", { brand: APP_BRAND_NAME })}
            </p>
          </div>
        </ChatEmptyState>
      </div>

      <LoginDialog open={loginOpen} onOpenChange={setLoginOpen} />
    </div>
  );
}
