/* eslint-disable no-nested-ternary */
export type SupportedLangs = 'az' | 'bo' | 'dz' | 'id' | 'ja' | 'jv' | 'ka' | 'km' | 'kn' | 'ko' | 'ms' | 'th' | 'tr' | 'vi' | 'zh' | 'af' | 'bn' | 'bg' | 'ca' | 'da' | 'de' | 'el' | 'en' | 'eo' | 'es' | 'et' | 'eu' | 'fa' | 'fi' | 'fo' | 'fur' | 'fy' | 'gl' | 'gu' | 'ha' | 'he' | 'hu' | 'is' | 'it' | 'ku' | 'lb' | 'ml' | 'mn' | 'mr' | 'nah' | 'nb' | 'ne' | 'nl' | 'nn' | 'no' | 'oc' | 'om' | 'or' | 'pa' | 'pap' | 'ps' | 'pt' | 'so' | 'sq' | 'sv' | 'sw' | 'ta' | 'te' | 'tk' | 'ur' | 'zu' | 'am' | 'bh' | 'fil' | 'fr' | 'gun' | 'hi' | 'hy' | 'ln' | 'mg' | 'nso' | 'xbr' | 'ti' | 'wa' | 'be' | 'bs' | 'hr' | 'ru' | 'sr' | 'uk' | 'cs' | 'sk' | 'ga' | 'lt' | 'sl' | 'mk' | 'mt' | 'lv' | 'pl' | 'cy' | 'ro' | 'ar';

export type GenericLocales = {
    [key in SupportedLangs]: SupportedLangs;
};

export enum AvailableLocales {
    az = 'az',
    bo = 'bo',
    dz = 'dz',
    id = 'id',
    ja = 'ja',
    jv = 'jv',
    ka = 'ka',
    km = 'km',
    kn = 'kn',
    ko = 'ko',
    ms = 'ms',
    th = 'th',
    tr = 'tr',
    vi = 'vi',
    zh = 'zh',
    af = 'af',
    bn = 'bn',
    bg = 'bg',
    ca = 'ca',
    da = 'da',
    de = 'de',
    el = 'el',
    en = 'en',
    eo = 'eo',
    es = 'es',
    et = 'et',
    eu = 'eu',
    fa = 'fa',
    fi = 'fi',
    fo = 'fo',
    fur = 'fur',
    fy = 'fy',
    gl = 'gl',
    gu = 'gu',
    ha = 'ha',
    he = 'he',
    hu = 'hu',
    is = 'is',
    it = 'it',
    ku = 'ku',
    lb = 'lb',
    ml = 'ml',
    mn = 'mn',
    mr = 'mr',
    nah = 'nah',
    nb = 'nb',
    ne = 'ne',
    nl = 'nl',
    nn = 'nn',
    no = 'no',
    oc = 'oc',
    om = 'om',
    or = 'or',
    pa = 'pa',
    pap = 'pap',
    ps = 'ps',
    pt = 'pt',
    so = 'so',
    sq = 'sq',
    sv = 'sv',
    sw = 'sw',
    ta = 'ta',
    te = 'te',
    tk = 'tk',
    ur = 'ur',
    zu = 'zu',
    am = 'am',
    bh = 'bh',
    fil = 'fil',
    fr = 'fr',
    gun = 'gun',
    hi = 'hi',
    hy = 'hy',
    ln = 'ln',
    mg = 'mg',
    nso = 'nso',
    xbr = 'xbr',
    ti = 'ti',
    wa = 'wa',
    be = 'be',
    bs = 'bs',
    hr = 'hr',
    ru = 'ru',
    sr = 'sr',
    uk = 'uk',
    cs = 'cs',
    sk = 'sk',
    ga = 'ga',
    lt = 'lt',
    sl = 'sl',
    mk = 'mk',
    mt = 'mt',
    lv = 'lv',
    pl = 'pl',
    cy = 'cy',
    ro = 'ro',
    ar = 'ar',
}
export const getPluralFormId = (locale: AvailableLocales, number: number) => {
    if (number === 0) {
        return 0;
    }
    const slavNum = ((number % 10 === 1) && (number % 100 !== 11))
        ? 1
        : (
            ((number % 10 >= 2) && (number % 10 <= 4) && ((number % 100 < 10)
            || (number % 100 >= 20))
            )
                ? 2
                : 3);
    const supportedForms: Record<AvailableLocales, number> = {
        [AvailableLocales.az]: 1,
        [AvailableLocales.bo]: 1,
        [AvailableLocales.dz]: 1,
        [AvailableLocales.id]: 1,
        [AvailableLocales.ja]: 1,
        [AvailableLocales.jv]: 1,
        [AvailableLocales.ka]: 1,
        [AvailableLocales.km]: 1,
        [AvailableLocales.kn]: 1,
        [AvailableLocales.ko]: 1,
        [AvailableLocales.ms]: 1,
        [AvailableLocales.th]: 1,
        [AvailableLocales.tr]: 1,
        [AvailableLocales.vi]: 1,
        [AvailableLocales.zh]: 1,

        [AvailableLocales.af]: (number === 1) ? 1 : 2,
        [AvailableLocales.bn]: (number === 1) ? 1 : 2,
        [AvailableLocales.bg]: (number === 1) ? 1 : 2,
        [AvailableLocales.ca]: (number === 1) ? 1 : 2,
        [AvailableLocales.da]: (number === 1) ? 1 : 2,
        [AvailableLocales.de]: (number === 1) ? 1 : 2,
        [AvailableLocales.el]: (number === 1) ? 1 : 2,
        [AvailableLocales.en]: (number === 1) ? 1 : 2,
        [AvailableLocales.eo]: (number === 1) ? 1 : 2,
        [AvailableLocales.es]: (number === 1) ? 1 : 2,
        [AvailableLocales.et]: (number === 1) ? 1 : 2,
        [AvailableLocales.eu]: (number === 1) ? 1 : 2,
        [AvailableLocales.fa]: (number === 1) ? 1 : 2,
        [AvailableLocales.fi]: (number === 1) ? 1 : 2,
        [AvailableLocales.fo]: (number === 1) ? 1 : 2,
        [AvailableLocales.fur]: (number === 1) ? 1 : 2,
        [AvailableLocales.fy]: (number === 1) ? 1 : 2,
        [AvailableLocales.gl]: (number === 1) ? 1 : 2,
        [AvailableLocales.gu]: (number === 1) ? 1 : 2,
        [AvailableLocales.ha]: (number === 1) ? 1 : 2,
        [AvailableLocales.he]: (number === 1) ? 1 : 2,
        [AvailableLocales.hu]: (number === 1) ? 1 : 2,
        [AvailableLocales.is]: (number === 1) ? 1 : 2,
        [AvailableLocales.it]: (number === 1) ? 1 : 2,
        [AvailableLocales.ku]: (number === 1) ? 1 : 2,
        [AvailableLocales.lb]: (number === 1) ? 1 : 2,
        [AvailableLocales.ml]: (number === 1) ? 1 : 2,
        [AvailableLocales.mn]: (number === 1) ? 1 : 2,
        [AvailableLocales.mr]: (number === 1) ? 1 : 2,
        [AvailableLocales.nah]: (number === 1) ? 1 : 2,
        [AvailableLocales.nb]: (number === 1) ? 1 : 2,
        [AvailableLocales.ne]: (number === 1) ? 1 : 2,
        [AvailableLocales.nl]: (number === 1) ? 1 : 2,
        [AvailableLocales.nn]: (number === 1) ? 1 : 2,
        [AvailableLocales.no]: (number === 1) ? 1 : 2,
        [AvailableLocales.oc]: (number === 1) ? 1 : 2,
        [AvailableLocales.om]: (number === 1) ? 1 : 2,
        [AvailableLocales.or]: (number === 1) ? 1 : 2,
        [AvailableLocales.pa]: (number === 1) ? 1 : 2,
        [AvailableLocales.pap]: (number === 1) ? 1 : 2,
        [AvailableLocales.ps]: (number === 1) ? 1 : 2,
        [AvailableLocales.pt]: (number === 1) ? 1 : 2,
        [AvailableLocales.so]: (number === 1) ? 1 : 2,
        [AvailableLocales.sq]: (number === 1) ? 1 : 2,
        [AvailableLocales.sv]: (number === 1) ? 1 : 2,
        [AvailableLocales.sw]: (number === 1) ? 1 : 2,
        [AvailableLocales.ta]: (number === 1) ? 1 : 2,
        [AvailableLocales.te]: (number === 1) ? 1 : 2,
        [AvailableLocales.tk]: (number === 1) ? 1 : 2,
        [AvailableLocales.ur]: (number === 1) ? 1 : 2,
        [AvailableLocales.zu]: (number === 1) ? 1 : 2,

        // how it works with 0?
        [AvailableLocales.am]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.bh]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.fil]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.fr]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.gun]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.hi]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.hy]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.ln]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.mg]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.nso]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.xbr]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.ti]: ((number === 0) || (number === 1)) ? 0 : 1,
        [AvailableLocales.wa]: ((number === 0) || (number === 1)) ? 0 : 1,

        [AvailableLocales.be]: slavNum,
        [AvailableLocales.bs]: slavNum,
        [AvailableLocales.hr]: slavNum,
        [AvailableLocales.ru]: slavNum,
        [AvailableLocales.sr]: slavNum,
        [AvailableLocales.uk]: slavNum,

        [AvailableLocales.cs]: (number === 1) ? 1 : (((number >= 2) && (number <= 4)) ? 2 : 3),
        [AvailableLocales.sk]: (number === 1) ? 1 : (((number >= 2) && (number <= 4)) ? 2 : 3),
        [AvailableLocales.ga]: (number === 1) ? 1 : ((number === 2) ? 2 : 3),
        [AvailableLocales.lt]: ((number % 10 === 1) && (number % 100 !== 11))
            ? 1
            : (((number % 10 >= 2) && ((number % 100 < 10) || (number % 100 >= 20)))
                ? 2
                : 3),
        [AvailableLocales.sl]: (number % 100 === 1)
            ? 1
            : ((number % 100 === 2)
                ? 2
                : (((number % 100 === 3) || (number % 100 === 4))
                    ? 3
                    : 4)),
        [AvailableLocales.mk]: (number % 10 === 1) ? 1 : 2,
        [AvailableLocales.mt]: (number === 1)
            ? 1
            : (((number === 0) || ((number % 100 > 1) && (number % 100 < 11)))
                ? 2
                : (((number % 100 > 10) && (number % 100 < 20))
                    ? 3
                    : 4)),
        [AvailableLocales.lv]: (number === 0)
            ? 0
            : (((number % 10 === 1) && (number % 100 !== 11))
                ? 1
                : 2),
        [AvailableLocales.pl]: (number === 1)
            ? 1
            : (
                ((number % 10 >= 2) && (number % 10 <= 4) && ((number % 100 < 12)
                || (number % 100 > 14))
                )
                    ? 2
                    : 3),
        [AvailableLocales.cy]: (number === 1)
            ? 0
            : ((number === 2)
                ? 1
                : (((number === 8) || (number === 11))
                    ? 2
                    : 3)),
        [AvailableLocales.ro]: (number === 1)
            ? 1
            : (((number === 1) || ((number % 100 > 0) && (number % 100 < 20)))
                ? 2
                : 3),
        [AvailableLocales.ar]: (number === 0)
            ? 0
            : ((number === 1)
                ? 1
                : ((number === 2)
                    ? 2
                    : (((number % 100 >= 3) && (number % 100 <= 10))
                        ? 3
                        : (((number % 100 >= 11) && (number % 100 <= 99))
                            ? 4
                            : 5)))),

    };
    return supportedForms[locale];
};
export const pluraFormsCount: Record<AvailableLocales, number> = {
    [AvailableLocales.az]: 2,
    [AvailableLocales.bo]: 2,
    [AvailableLocales.dz]: 2,
    [AvailableLocales.id]: 2,
    [AvailableLocales.ja]: 2,
    [AvailableLocales.jv]: 2,
    [AvailableLocales.ka]: 2,
    [AvailableLocales.km]: 2,
    [AvailableLocales.kn]: 2,
    [AvailableLocales.ko]: 2,
    [AvailableLocales.ms]: 2,
    [AvailableLocales.th]: 2,
    [AvailableLocales.tr]: 2,
    [AvailableLocales.vi]: 2,
    [AvailableLocales.zh]: 2,
    [AvailableLocales.af]: 3,
    [AvailableLocales.bn]: 3,
    [AvailableLocales.bg]: 3,
    [AvailableLocales.ca]: 3,
    [AvailableLocales.da]: 3,
    [AvailableLocales.de]: 3,
    [AvailableLocales.el]: 3,
    [AvailableLocales.en]: 3,
    [AvailableLocales.eo]: 3,
    [AvailableLocales.es]: 3,
    [AvailableLocales.et]: 3,
    [AvailableLocales.eu]: 3,
    [AvailableLocales.fa]: 3,
    [AvailableLocales.fi]: 3,
    [AvailableLocales.fo]: 3,
    [AvailableLocales.fur]: 3,
    [AvailableLocales.fy]: 3,
    [AvailableLocales.gl]: 3,
    [AvailableLocales.gu]: 3,
    [AvailableLocales.ha]: 3,
    [AvailableLocales.he]: 3,
    [AvailableLocales.hu]: 3,
    [AvailableLocales.is]: 3,
    [AvailableLocales.it]: 3,
    [AvailableLocales.ku]: 3,
    [AvailableLocales.lb]: 3,
    [AvailableLocales.ml]: 3,
    [AvailableLocales.mn]: 3,
    [AvailableLocales.mr]: 3,
    [AvailableLocales.nah]: 3,
    [AvailableLocales.nb]: 3,
    [AvailableLocales.ne]: 3,
    [AvailableLocales.nl]: 3,
    [AvailableLocales.nn]: 3,
    [AvailableLocales.no]: 3,
    [AvailableLocales.oc]: 3,
    [AvailableLocales.om]: 3,
    [AvailableLocales.or]: 3,
    [AvailableLocales.pa]: 3,
    [AvailableLocales.pap]: 3,
    [AvailableLocales.ps]: 3,
    [AvailableLocales.pt]: 3,
    [AvailableLocales.so]: 3,
    [AvailableLocales.sq]: 3,
    [AvailableLocales.sv]: 3,
    [AvailableLocales.sw]: 3,
    [AvailableLocales.ta]: 3,
    [AvailableLocales.te]: 3,
    [AvailableLocales.tk]: 3,
    [AvailableLocales.ur]: 3,
    [AvailableLocales.zu]: 3,
    [AvailableLocales.am]: 2,
    [AvailableLocales.bh]: 2,
    [AvailableLocales.fil]: 2,
    [AvailableLocales.fr]: 2,
    [AvailableLocales.gun]: 2,
    [AvailableLocales.hi]: 2,
    [AvailableLocales.hy]: 2,
    [AvailableLocales.ln]: 2,
    [AvailableLocales.mg]: 2,
    [AvailableLocales.nso]: 2,
    [AvailableLocales.xbr]: 2,
    [AvailableLocales.ti]: 2,
    [AvailableLocales.wa]: 2,
    [AvailableLocales.be]: 4,
    [AvailableLocales.bs]: 4,
    [AvailableLocales.hr]: 4,
    [AvailableLocales.ru]: 4,
    [AvailableLocales.sr]: 4,
    [AvailableLocales.uk]: 4,
    [AvailableLocales.cs]: 4,
    [AvailableLocales.sk]: 4,
    [AvailableLocales.ga]: 4,
    [AvailableLocales.lt]: 4,
    [AvailableLocales.sl]: 5,
    [AvailableLocales.mk]: 3,
    [AvailableLocales.mt]: 5,
    [AvailableLocales.lv]: 3,
    [AvailableLocales.pl]: 4,
    [AvailableLocales.cy]: 4,
    [AvailableLocales.ro]: 4,
    [AvailableLocales.ar]: 6,
};

const PLURAL_STRING_DELIMITER = '|';

export const checkForms = (str: string, locale: AvailableLocales, id: string) => {
    const forms = str.split(PLURAL_STRING_DELIMITER);
    if (forms.length !== pluraFormsCount[locale]) {
        throw new Error(`Invalid plural string "${id}" for locale ${locale}: ${forms.length} given; need: ${pluraFormsCount[locale]}`);
    }
};
export const checkFormsExternal = (str: string, locale: AvailableLocales, id: string) => {
    try {
        checkForms(str, locale, id);
        return true;
    } catch (error) {
        return false;
    }
};
export const getForm = (str: string, number: number, locale: AvailableLocales, id: string) => {
    checkForms(str, locale, id);
    const forms = str.split(PLURAL_STRING_DELIMITER);
    const currentForm = getPluralFormId(locale, number);
    return forms[currentForm].trim();
};
