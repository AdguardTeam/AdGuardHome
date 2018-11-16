import i18n from 'i18next';
import { reactI18nextModule } from 'react-i18next';
import { initReactI18n } from 'react-i18next/hooks';
import langDetect from 'i18next-browser-languagedetector';

import vi from './__locales/vi.json';
import en from './__locales/en.json';

export const languages = [
    {
        key: 'en',
        name: 'English',
    },
    {
        key: 'vi',
        name: 'Tiếng Việt',
    },
    {
        key: 'ru',
        name: 'Русский',
    },
];

i18n
    .use(langDetect)
    .use(initReactI18n)
    .use(reactI18nextModule) // passes i18n down to react-i18next
    .init({
        resources: {
            vi: {
                translation: vi,
            },
            en: {
                translation: en,
            },
        },
        fallbackLng: 'en',
        keySeparator: false, // we use content as keys
        nsSeparator: false, // Fix character in content
        interpolation: {
            escapeValue: false, // not needed for react!!
        },
        react: {
            wait: true,
        },
    });

export default i18n;
