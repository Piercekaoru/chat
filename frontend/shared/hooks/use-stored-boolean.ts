"use client"

import * as React from "react"

import { readLocalStorageItem, writeLocalStorageItem } from "@/shared/lib/storage-key-migration"

export function useStoredBoolean(storageKey: string, defaultValue: boolean, legacyStorageKey?: string) {
  const [value, setValue] = React.useState(() => {
    if (typeof window === "undefined") {
      return defaultValue
    }

    const stored = readLocalStorageItem(storageKey, legacyStorageKey)
    if (stored === "true") {
      return true
    }
    if (stored === "false") {
      return false
    }

    return defaultValue
  })

  React.useEffect(() => {
    writeLocalStorageItem(storageKey, value ? "true" : "false", legacyStorageKey)
  }, [legacyStorageKey, storageKey, value])

  return [value, setValue] as const
}
