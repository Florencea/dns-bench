import { useSyncExternalStore } from "react";
import type { SupportedLocale } from "../../shared/locales";

let currentLocale: SupportedLocale = "zh-TW";
const listeners = new Set<() => void>();

function notify(): void {
  for (const listener of listeners) {
    listener();
  }
}

export function initLocaleSubscription(): void {
  if (window.api !== undefined) {
    const { getLocale, onLocaleChange } = window.api;
    const fetchInitial = async (): Promise<void> => {
      const saved = await getLocale();
      if (currentLocale !== saved) {
        currentLocale = saved;
        notify();
      }
    };
    void fetchInitial();

    onLocaleChange((newLocale) => {
      if (currentLocale !== newLocale) {
        currentLocale = newLocale;
        notify();
      }
    });
  }
}

export function resetLocaleStore(initial: SupportedLocale = "zh-TW"): void {
  currentLocale = initial;
  notify();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function getSnapshot(): SupportedLocale {
  return currentLocale;
}

function getServerSnapshot(): SupportedLocale {
  return "zh-TW";
}

async function changeLocale(newLocale: SupportedLocale): Promise<void> {
  if (currentLocale !== newLocale) {
    currentLocale = newLocale;
    notify();
  }
  if (window.api !== undefined) {
    await window.api.setLocale(newLocale);
  }
}

export function useLocale(): [
  SupportedLocale,
  (newLocale: SupportedLocale) => Promise<void>,
] {
  const locale = useSyncExternalStore(
    subscribe,
    getSnapshot,
    getServerSnapshot,
  );
  return [locale, changeLocale];
}
