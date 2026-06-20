import i18n from "i18next";
import { initReactI18next, } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import en from "./locales/en/translation.json";
import zhHans from "./locales/zh-Hans/translation.json";
import ja from "./locales/ja/translation.json";
import ko from "./locales/ko/translation.json";
import fr from "./locales/fr/translation.json";
import it from "./locales/it/translation.json";
import ru from "./locales/ru/translation.json";
import nl from "./locales/nl/translation.json";

i18n
  .use(LanguageDetector,)
  .use(initReactI18next,)
  .init({
    resources: {
      en: { translation: en, },
      "zh-Hans": { translation: zhHans, },
      ja: { translation: ja, },
      ko: { translation: ko, },
      fr: { translation: fr, },
      it: { translation: it, },
      ru: { translation: ru, },
      nl: { translation: nl, },
    },
    fallbackLng: {
      "zh-CN": ["zh-Hans",],
      "zh-SG": ["zh-Hans",],
      "zh-TW": ["zh-Hant",],
      "zh-HK": ["zh-Hant",],
      zh: ["zh-Hans",],
      default: ["en",],
    },
    detection: {
      order: ["localStorage", "navigator",],
      caches: ["localStorage",],
    },
    interpolation: {
      escapeValue: false,
    },
  },);
