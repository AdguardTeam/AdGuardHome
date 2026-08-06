import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import langDetect from 'i18next-browser-languagedetector';
import { setHtmlLangAttr } from './helpers/helpers';

import { LANGUAGES, BASE_LOCALE } from './helpers/twosky';

// Main translations
import ar from './__locales/ar.json';
import be from './__locales/be.json';
import bg from './__locales/bg.json';
import cs from './__locales/cs.json';
import da from './__locales/da.json';
import de from './__locales/de.json';
import en from './__locales/en.json';
import es from './__locales/es.json';
import fa from './__locales/fa.json';
import fi from './__locales/fi.json';
import fr from './__locales/fr.json';
import hr from './__locales/hr.json';
import hu from './__locales/hu.json';
import id from './__locales/id.json';
import it from './__locales/it.json';
import ja from './__locales/ja.json';
import ko from './__locales/ko.json';
import nl from './__locales/nl.json';
import no from './__locales/no.json';
import pl from './__locales/pl.json';
import ptBR from './__locales/pt-br.json';
import ptPT from './__locales/pt-pt.json';
import ro from './__locales/ro.json';
import ru from './__locales/ru.json';
import siLk from './__locales/si-lk.json';
import sk from './__locales/sk.json';
import sl from './__locales/sl.json';
import srCS from './__locales/sr-cs.json';
import sv from './__locales/sv.json';
import th from './__locales/th.json';
import tr from './__locales/tr.json';
import uk from './__locales/uk.json';
import vi from './__locales/vi.json';
import zhCN from './__locales/zh-cn.json';
import zhHK from './__locales/zh-hk.json';
import zhTW from './__locales/zh-tw.json';

// Services translations
import arServices from './__locales-services/ar.json';
import beServices from './__locales-services/be.json';
import bgServices from './__locales-services/bg.json';
import csServices from './__locales-services/cs.json';
import daServices from './__locales-services/da.json';
import deServices from './__locales-services/de.json';
import enServices from './__locales-services/en.json';
import esServices from './__locales-services/es.json';
import faServices from './__locales-services/fa.json';
import fiServices from './__locales-services/fi.json';
import frServices from './__locales-services/fr.json';
import hrServices from './__locales-services/hr.json';
import huServices from './__locales-services/hu.json';
import idServices from './__locales-services/id.json';
import itServices from './__locales-services/it.json';
import jaServices from './__locales-services/ja.json';
import koServices from './__locales-services/ko.json';
import nlServices from './__locales-services/nl.json';
import noServices from './__locales-services/no.json';
import plServices from './__locales-services/pl.json';
import ptBRServices from './__locales-services/pt-br.json';
import ptPTServices from './__locales-services/pt-pt.json';
import roServices from './__locales-services/ro.json';
import ruServices from './__locales-services/ru.json';
import siLkServices from './__locales-services/si-lk.json';
import skServices from './__locales-services/sk.json';
import slServices from './__locales-services/sl.json';
import srCSServices from './__locales-services/sr-cs.json';
import svServices from './__locales-services/sv.json';
import thServices from './__locales-services/th.json';
import trServices from './__locales-services/tr.json';
import ukServices from './__locales-services/uk.json';
import viServices from './__locales-services/vi.json';
import zhCNServices from './__locales-services/zh-cn.json';
import zhHKServices from './__locales-services/zh-hk.json';
import zhTWServices from './__locales-services/zh-tw.json';

/**
 * Helper function to convert services object into a flat `{ key: message }` format.
 *
 * Supported formats:
 * - { message: "..." }
 *
 * Example:
 * Input:  { a: { message: "one" }, b: { message: "two" } }
 * Output: { a: "one", b: "two" }
 */
const convertServicesFormat = (
    services: Record<string, { message: string }>,
): Record<string, string> => {
    return Object.fromEntries(
        Object.entries(services).map(([key, value]) => [key, value.message])
    );
};

// Resources
const resources = {
    ar: {
        translation: ar,
        services: convertServicesFormat(arServices)
    },
    be: {
        translation: be,
        services: convertServicesFormat(beServices)
    },
    bg: {
        translation: bg,
        services: convertServicesFormat(bgServices)
    },
    cs: {
        translation: cs,
        services: convertServicesFormat(csServices)
    },
    da: {
        translation: da,
        services: convertServicesFormat(daServices)
    },
    de: {
        translation: de,
        services: convertServicesFormat(deServices)
    },
    en: {
        translation: en,
        services: convertServicesFormat(enServices)
    },
    'en-us': {
        translation: en,
        services: convertServicesFormat(enServices)
    },
    es: {
        translation: es,
        services: convertServicesFormat(esServices)
    },
    fa: {
        translation: fa,
        services: convertServicesFormat(faServices)
    },
    fi: {
        translation: fi,
        services: convertServicesFormat(fiServices)
    },
    fr: {
        translation: fr,
        services: convertServicesFormat(frServices)
    },
    hr: {
        translation: hr,
        services: convertServicesFormat(hrServices)
    },
    hu: {
        translation: hu,
        services: convertServicesFormat(huServices)
    },
    id: {
        translation: id,
        services: convertServicesFormat(idServices)
    },
    it: {
        translation: it,
        services: convertServicesFormat(itServices)
    },
    ja: {
        translation: ja,
        services: convertServicesFormat(jaServices)
    },
    ko: {
        translation: ko,
        services: convertServicesFormat(koServices)
    },
    nl: {
        translation: nl,
        services: convertServicesFormat(nlServices)
    },
    no: {
        translation: no,
        services: convertServicesFormat(noServices)
    },
    pl: {
        translation: pl,
        services: convertServicesFormat(plServices)
    },
    'pt-br': {
        translation: ptBR,
        services: convertServicesFormat(ptBRServices)
    },
    'pt-pt': {
        translation: ptPT,
        services: convertServicesFormat(ptPTServices)
    },
    ro: {
        translation: ro,
        services: convertServicesFormat(roServices)
    },
    ru: {
        translation: ru,
        services: convertServicesFormat(ruServices)
    },
    'si-lk': {
        translation: siLk,
        services: convertServicesFormat(siLkServices)
    },
    sk: {
        translation: sk,
        services: convertServicesFormat(skServices)
    },
    sl: {
        translation: sl,
        services: convertServicesFormat(slServices)
    },
    'sr-cs': {
        translation: srCS,
        services: convertServicesFormat(srCSServices)
    },
    sv: {
        translation: sv,
        services: convertServicesFormat(svServices)
    },
    th: {
        translation: th,
        services: convertServicesFormat(thServices)
    },
    tr: {
        translation: tr,
        services: convertServicesFormat(trServices)
    },
    uk: {
        translation: uk,
        services: convertServicesFormat(ukServices)
    },
    vi: {
        translation: vi,
        services: convertServicesFormat(viServices)
    },
    'zh-cn': {
        translation: zhCN,
        services: convertServicesFormat(zhCNServices)
    },
    'zh-hk': {
        translation: zhHK,
        services: convertServicesFormat(zhHKServices)
    },
    'zh-tw': {
        translation: zhTW,
        services: convertServicesFormat(zhTWServices)
    },
};

const availableLanguages = Object.keys(LANGUAGES);

i18n
    .use(langDetect)
    .use(initReactI18next)
    .init(
        {
            resources,
            lowerCaseLng: true,
            fallbackLng: BASE_LOCALE,
            keySeparator: false,
            nsSeparator: false,
            returnEmptyString: false,
            ns: ['translation', 'services'],
            defaultNS: 'translation',
            interpolation: {
                escapeValue: false,
            },
            react: {
                wait: true,
                bindI18n: 'languageChanged loaded',
            },
            whitelist: availableLanguages,
        },
        () => {
            if (!availableLanguages.includes(i18n.language)) {
                i18n.changeLanguage(BASE_LOCALE);
            }
            setHtmlLangAttr(i18n.language);
        }
    );

i18n.on('languageChanged', (lng) => {
    setHtmlLangAttr(lng);
});

export default i18n;
