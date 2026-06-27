export function migrateLocalStorageKey(storageKey: string, legacyStorageKey?: string): void {
  if (typeof window === "undefined" || !legacyStorageKey || storageKey === legacyStorageKey) {
    return;
  }

  try {
    const currentValue = window.localStorage.getItem(storageKey);
    const legacyValue = window.localStorage.getItem(legacyStorageKey);
    if (currentValue === null && legacyValue !== null) {
      window.localStorage.setItem(storageKey, legacyValue);
    }
    if (legacyValue !== null) {
      window.localStorage.removeItem(legacyStorageKey);
    }
  } catch {
    // localStorage can be unavailable in private browsing or strict environments.
  }
}

export function readLocalStorageItem(storageKey: string, legacyStorageKey?: string): string | null {
  if (typeof window === "undefined") {
    return null;
  }

  migrateLocalStorageKey(storageKey, legacyStorageKey);
  try {
    return window.localStorage.getItem(storageKey);
  } catch {
    return null;
  }
}

export function writeLocalStorageItem(storageKey: string, value: string, legacyStorageKey?: string): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, value);
    if (legacyStorageKey && legacyStorageKey !== storageKey) {
      window.localStorage.removeItem(legacyStorageKey);
    }
  } catch {
    // localStorage can be unavailable in private browsing or strict environments.
  }
}

export function removeLocalStorageItem(storageKey: string, legacyStorageKey?: string): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.removeItem(storageKey);
    if (legacyStorageKey && legacyStorageKey !== storageKey) {
      window.localStorage.removeItem(legacyStorageKey);
    }
  } catch {
    // localStorage can be unavailable in private browsing or strict environments.
  }
}

export function storageEventMatchesKey(event: StorageEvent, storageKey: string, legacyStorageKey?: string): boolean {
  return event.key === storageKey || (!!legacyStorageKey && event.key === legacyStorageKey);
}
