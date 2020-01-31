import i18n from 'i18next';
import XHR from 'i18next-xhr-backend';
import { reactI18nextModule } from 'react-i18next';
import { initReactI18n } from 'react-i18next/hooks';
import langDetect from 'i18next-browser-languagedetector';

import { LANGUAGES, BASE_LOCALE } from './helpers/twosky';

const availableLanguages = Object.keys(LANGUAGES);

i18n
    .use(langDetect)
    .use(XHR)
    .use(initReactI18n)
    .use(reactI18nextModule)
    .init({
        lowerCaseLng: true,
        fallbackLng: BASE_LOCALE,
        keySeparator: false,
        nsSeparator: false,
        returnEmptyString: false,
        interpolation: {
            escapeValue: false,
        },
        react: {
            wait: true,
        },
        whitelist: availableLanguages,
        backend: {
            loadPath: '/__locales/{{lng}}.json',
        },
    }, () => {
        if (!availableLanguages.includes(i18n.language)) {
            i18n.changeLanguage(BASE_LOCALE);
        }
    });

export default i18n;
