import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import langDetect from 'i18next-browser-languagedetector';

import { LANGUAGES, BASE_LOCALE } from './helpers/twosky';

import vi from './__locales/vi.json';
import en from './__locales/en.json';
import ru from './__locales/ru.json';
import es from './__locales/es.json';
import fr from './__locales/fr.json';
import ja from './__locales/ja.json';
import sv from './__locales/sv.json';
import ptBR from './__locales/pt-br.json';
import zhTW from './__locales/zh-tw.json';
import bg from './__locales/bg.json';
import zhCN from './__locales/zh-cn.json';
import cs from './__locales/cs.json';
import da from './__locales/da.json';
import de from './__locales/de.json';
import id from './__locales/id.json';
import it from './__locales/it.json';
import ko from './__locales/ko.json';
import no from './__locales/no.json';
import nl from './__locales/nl.json';
import pl from './__locales/pl.json';
import ptPT from './__locales/pt-pt.json';
import sk from './__locales/sk.json';
import sl from './__locales/sl.json';
import tr from './__locales/tr.json';
import srCS from './__locales/sr-cs.json';
import hr from './__locales/hr.json';
import hu from './__locales/hu.json';
import fa from './__locales/fa.json';
import th from './__locales/th.json';
import ro from './__locales/ro.json';
import siLk from './__locales/si-lk.json';
import { setHtmlLangAttr } from './helpers/helpers';

const resources = {
    en: {
        translation: en,
    },
    vi: {
        translation: vi,
    },
    ru: {
        translation: ru,
    },
    es: {
        translation: es,
    },
    fr: {
        translation: fr,
    },
    ja: {
        translation: ja,
    },
    sv: {
        translation: sv,
    },
    'pt-br': {
        translation: ptBR,
    },
    'zh-tw': {
        translation: zhTW,
    },
    bg: {
        translation: bg,
    },
    'zh-cn': {
        translation: zhCN,
    },
    cs: {
        translation: cs,
    },
    da: {
        translation: da,
    },
    de: {
        translation: de,
    },
    id: {
        translation: id,
    },
    it: {
        translation: it,
    },
    ko: {
        translation: ko,
    },
    no: {
        translation: no,
    },
    nl: {
        translation: nl,
    },
    pl: {
        translation: pl,
    },
    'pt-pt': {
        translation: ptPT,
    },
    sk: {
        translation: sk,
    },
    sl: {
        translation: sl,
    },
    tr: {
        translation: tr,
    },
    'sr-cs': {
        translation: srCS,
    },
    hr: {
        translation: hr,
    },
    hu: {
        translation: hu,
    },
    fa: {
        translation: fa,
    },
    th: {
        translation: th,
    },
    ro: {
        translation: ro,
    },
    'si-lk': {
        translation: siLk,
    },
};

const availableLanguages = Object.keys(LANGUAGES);

i18n
    .use(langDetect)
    .use(initReactI18next)
    .init({
        resources,
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
    },
    () => {
        if (!availableLanguages.includes(i18n.language)) {
            i18n.changeLanguage(BASE_LOCALE);
        }
        setHtmlLangAttr(i18n.language);
    });

export default i18n;
