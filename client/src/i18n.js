
import i18n from 'i18next';
import { reactI18nextModule } from 'react-i18next';
import langDetect from 'i18next-browser-languagedetector';
import viResource from './__locales/vi';

i18n
    .use(langDetect)
    .use(reactI18nextModule) // passes i18n down to react-i18next
    .init({
        resources: {
            vi: viResource,
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
