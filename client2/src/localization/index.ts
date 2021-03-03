import { Locale, DatePickerLocale, messages, DEFAULT_LOCALE, LANGUAGES } from './locales';

export { Locale, DatePickerLocale, messages, DEFAULT_LOCALE, LANGUAGES };
export const i18n = (lang: Locale) => ({
    getMessage: (key: string) => messages[lang][key],
    getUILanguage: () => lang,
    getBaseMessage: (key: string) => messages[DEFAULT_LOCALE][key] || key,
    getBaseUILanguage: () => DEFAULT_LOCALE,
});
