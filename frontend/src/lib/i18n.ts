import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import {
  LANGUAGE_STORAGE_KEY,
  normalizeLanguage,
  readStoredLanguage,
  supportedLanguages,
  type SupportedLanguage,
} from "@/lib/preferences";
import zhCN from "../locales/zh-CN";
import enUS from "../locales/en-US";

type LocaleResource = {
  translation: Record<string, unknown>;
};

const localeResources: Record<SupportedLanguage, LocaleResource> = {
  "zh-CN": zhCN,
  "en-US": enUS,
};

const loadedLanguages = new Set<SupportedLanguage>();
let initializationPromise: Promise<typeof i18n> | null = null;
let languageListenerAttached = false;

function getLocaleResource(language: SupportedLanguage): LocaleResource {
  const resource = localeResources[normalizeLanguage(language)];

  if (
    resource &&
    typeof resource === "object" &&
    resource.translation &&
    typeof resource.translation === "object"
  ) {
    return resource;
  }

  return zhCN;
}

function toPlainTranslationResource(language: SupportedLanguage) {
  const resource = getLocaleResource(language);

  return JSON.parse(JSON.stringify(resource.translation)) as Record<string, unknown>;
}

function syncDocumentLanguage(language: SupportedLanguage) {
  if (typeof document === "undefined") {
    return;
  }

  document.documentElement.lang = language;

  if (typeof window !== "undefined") {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  }
}

async function loadLanguageResource(language: SupportedLanguage) {
  if (loadedLanguages.has(language)) {
    return;
  }

  const messages = toPlainTranslationResource(language);
  i18n.addResourceBundle(language, "translation", messages, true, true);
  loadedLanguages.add(language);
}

function trimInactiveLanguageResources(activeLanguage: SupportedLanguage) {
  for (const language of supportedLanguages) {
    if (language === activeLanguage) {
      continue;
    }

    if (i18n.hasResourceBundle(language, "translation")) {
      i18n.removeResourceBundle(language, "translation");
    }

    loadedLanguages.delete(language);
  }
}

function attachLanguageListener() {
  if (languageListenerAttached) {
    return;
  }

  i18n.on("languageChanged", (language) => {
    syncDocumentLanguage(language as SupportedLanguage);
  });

  languageListenerAttached = true;
}

export async function initializeI18n() {
  if (i18n.isInitialized) {
    const initialLanguage = readStoredLanguage();

    trimInactiveLanguageResources(initialLanguage);

    if (i18n.hasResourceBundle(initialLanguage, "translation")) {
      loadedLanguages.add(initialLanguage);
    } else {
      await loadLanguageResource(initialLanguage);
    }

    if ((i18n.resolvedLanguage ?? i18n.language) !== initialLanguage) {
      await i18n.changeLanguage(initialLanguage);
    }

    attachLanguageListener();
    syncDocumentLanguage(initialLanguage);
    return i18n;
  }

  if (initializationPromise) {
    return initializationPromise;
  }

  initializationPromise = (async () => {
    const initialLanguage = readStoredLanguage();
    const initialMessages = toPlainTranslationResource(initialLanguage);

    loadedLanguages.add(initialLanguage);

    await i18n.use(initReactI18next).init({
      resources: {
        [initialLanguage]: {
          translation: initialMessages,
        },
      },
      lng: initialLanguage,
      fallbackLng: "zh-CN",
      supportedLngs: [...supportedLanguages],
      interpolation: {
        escapeValue: false,
      },
    });

    attachLanguageListener();
    syncDocumentLanguage(initialLanguage);

    return i18n;
  })();

  return initializationPromise;
}

export async function changeAppLanguage(language: SupportedLanguage) {
  const instance = await initializeI18n();

  await Promise.all(
    [...new Set<SupportedLanguage>([language, "zh-CN"])]
      .filter((item) => !loadedLanguages.has(item))
      .map((item) => loadLanguageResource(item)),
  );

  if ((instance.resolvedLanguage ?? instance.language) !== language) {
    await instance.changeLanguage(language);
  }

  syncDocumentLanguage(language);
  return instance;
}

export default i18n;
