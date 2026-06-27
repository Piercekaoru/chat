"use client";

import * as React from "react";

import {
  readLocalStorageItem,
  storageEventMatchesKey,
  writeLocalStorageItem,
} from "@/shared/lib/storage-key-migration";

export const FONT_SIZE_STORAGE_KEY = "openachieve:font-size";
const LEGACY_FONT_SIZE_STORAGE_KEY = "deeix-chat:font-size";
export const FONT_SIZE_UPDATED_EVENT = "openachieve:font-size-updated";

export type FontSizeOption = "small" | "standard" | "medium" | "large";

let currentFontSizePreference: FontSizeOption = "standard";
let fontSizePreferenceLoaded = false;

export function isFontSizeOption(value: unknown): value is FontSizeOption {
  return value === "small" || value === "standard" || value === "medium" || value === "large";
}

function getStoredFontSizePreference(): FontSizeOption {
  if (typeof window === "undefined") {
    return "standard";
  }

  const storedValue = readLocalStorageItem(FONT_SIZE_STORAGE_KEY, LEGACY_FONT_SIZE_STORAGE_KEY);
  return isFontSizeOption(storedValue) ? storedValue : "standard";
}

export function readFontSizePreference(): FontSizeOption {
  if (!fontSizePreferenceLoaded) {
    currentFontSizePreference = getStoredFontSizePreference();
    fontSizePreferenceLoaded = true;
  }

  return currentFontSizePreference;
}

export function applyFontSizePreference(value: FontSizeOption) {
  if (typeof document === "undefined") {
    return;
  }

  if (value === "standard") {
    delete document.documentElement.dataset.fontSize;
    return;
  }

  document.documentElement.dataset.fontSize = value;
}

function dispatchFontSizeUpdated(value: FontSizeOption) {
  if (typeof window === "undefined") {
    return;
  }

  window.dispatchEvent(new CustomEvent<FontSizeOption>(FONT_SIZE_UPDATED_EVENT, { detail: value }));
}

export function writeFontSizePreference(value: FontSizeOption) {
  currentFontSizePreference = value;
  fontSizePreferenceLoaded = true;

  if (typeof window !== "undefined") {
    writeLocalStorageItem(FONT_SIZE_STORAGE_KEY, value, LEGACY_FONT_SIZE_STORAGE_KEY);
  }

  applyFontSizePreference(value);
  dispatchFontSizeUpdated(value);
}

function subscribeFontSizePreference(onStoreChange: () => void) {
  if (typeof window === "undefined") {
    return () => undefined;
  }

  function handleStorage(event: StorageEvent) {
    if (!storageEventMatchesKey(event, FONT_SIZE_STORAGE_KEY, LEGACY_FONT_SIZE_STORAGE_KEY)) {
      return;
    }

    currentFontSizePreference = isFontSizeOption(event.newValue) ? event.newValue : "standard";
    fontSizePreferenceLoaded = true;
    applyFontSizePreference(currentFontSizePreference);
    onStoreChange();
  }

  function handleFontSizeUpdated() {
    onStoreChange();
  }

  window.addEventListener("storage", handleStorage);
  window.addEventListener(FONT_SIZE_UPDATED_EVENT, handleFontSizeUpdated);

  return () => {
    window.removeEventListener("storage", handleStorage);
    window.removeEventListener(FONT_SIZE_UPDATED_EVENT, handleFontSizeUpdated);
  };
}

export function useFontSizePreference() {
  return React.useSyncExternalStore(
    subscribeFontSizePreference,
    readFontSizePreference,
    (): FontSizeOption => "standard",
  );
}
