
import i18n from 'i18next';
import { reactI18nextModule } from 'react-i18next';
import { initReactI18n } from 'react-i18next/hooks';
import langDetect from 'i18next-browser-languagedetector';
import vi from './__locales/vi';
import en from './__locales/en';

export const languages = [
    {
        key: 'vi',
        name: 'Tiếng Việt',
    },
    {
        key: 'en',
        name: 'English',
    },
];


i18n
    .use(langDetect)
    .use(initReactI18n)
    .use(reactI18nextModule) // passes i18n down to react-i18next
    .init({
        resources: {
            vi,
            en,
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
