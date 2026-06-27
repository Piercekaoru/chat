"use client";

import { APP_BRAND_NAME } from "@/shared/brand";
import { readLocalStorageItem, writeLocalStorageItem } from "@/shared/lib/storage-key-migration";

const RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY = "openachieve:response-completion-notifications";
const LEGACY_RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY = "deeix-chat:response-completion-notifications";
const NOTIFICATION_BODY_MAX_LENGTH = 140;

type ResponseCompletionNotificationInput = {
  content?: string | null;
  conversationPublicID?: string | null;
  conversationTitle?: string | null;
};

function normalizeString(value: unknown, fallback = "") {
  if (typeof value !== "string") {
    return fallback;
  }

  const normalizedValue = value.trim();
  return normalizedValue || fallback;
}

function normalizeNotificationBody(value: string) {
  const compactValue = value.replace(/\s+/g, " ").trim();
  if (!compactValue) {
    return "Your reply is ready.";
  }
  if (compactValue.length <= NOTIFICATION_BODY_MAX_LENGTH) {
    return compactValue;
  }
  return `${compactValue.slice(0, NOTIFICATION_BODY_MAX_LENGTH - 1)}…`;
}

export function isBrowserNotificationSupported() {
  return typeof window !== "undefined" && "Notification" in window;
}

export function getBrowserNotificationPermission(): NotificationPermission | "unsupported" {
  if (!isBrowserNotificationSupported()) {
    return "unsupported";
  }
  return Notification.permission;
}

export function readResponseCompletionNotificationsEnabled() {
  if (typeof window === "undefined") {
    return false;
  }
  return readLocalStorageItem(
    RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY,
    LEGACY_RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY,
  ) === "true";
}

export function writeResponseCompletionNotificationsEnabled(enabled: boolean) {
  if (typeof window === "undefined") {
    return;
  }
  writeLocalStorageItem(
    RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY,
    String(enabled),
    LEGACY_RESPONSE_COMPLETION_NOTIFICATIONS_STORAGE_KEY,
  );
}

export async function enableResponseCompletionNotifications() {
  if (!isBrowserNotificationSupported()) {
    writeResponseCompletionNotificationsEnabled(false);
    return {
      enabled: false,
      permission: "unsupported" as const,
    };
  }

  let permission = Notification.permission;
  if (permission !== "granted") {
    permission = await Notification.requestPermission();
  }

  const enabled = permission === "granted";
  writeResponseCompletionNotificationsEnabled(enabled);

  return {
    enabled,
    permission,
  };
}

export function disableResponseCompletionNotifications() {
  writeResponseCompletionNotificationsEnabled(false);
}

export function notifyResponseCompletion(input: ResponseCompletionNotificationInput) {
  if (!isBrowserNotificationSupported()) {
    return false;
  }
  if (!readResponseCompletionNotificationsEnabled() || Notification.permission !== "granted") {
    return false;
  }
  if (typeof document !== "undefined" && document.visibilityState === "visible" && document.hasFocus()) {
    return false;
  }

  const conversationTitle = normalizeString(input.conversationTitle);
  const notification = new Notification(conversationTitle || APP_BRAND_NAME, {
    body: normalizeNotificationBody(normalizeString(input.content)),
    tag: normalizeString(input.conversationPublicID, `response-completion:${Date.now()}`),
    icon: "/favicon.ico",
  });

  notification.onclick = () => {
    if (typeof window !== "undefined") {
      void window.focus();
    }
    notification.close();
  };

  return true;
}
