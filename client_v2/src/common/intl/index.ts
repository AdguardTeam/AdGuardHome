import React from 'react';

import { I18nInterface, translate } from '@adguard/translate';
import { BASE_LOCALE } from 'panel/helpers/twosky';

import en from 'panel/__locales/en.json';
import de from 'panel/__locales/de.json';
import es from 'panel/__locales/es.json';
import fr from 'panel/__locales/fr.json';
import it from 'panel/__locales/it.json';
import ja from 'panel/__locales/ja.json';
import ko from 'panel/__locales/ko.json';
import ptBr from 'panel/__locales/pt-br.json';
import ptPt from 'panel/__locales/pt-pt.json';
import ru from 'panel/__locales/ru.json';
import zhCn from 'panel/__locales/zh-cn.json';
import zhTw from 'panel/__locales/zh-tw.json';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';

export type LocalesType = ReturnType<I18nInterface['getUILanguage']>;

type LocalesTypes = Partial<Record<LocalesType, Record<string, string>>>;

const LOCALES = {
    en,
    de,
    es,
    fr,
    it,
    ja,
    ko,
    ru,
    'pt-br': ptBr,
    'pt-pt': ptPt,
    'zh-cn': zhCn,
    'zh-tw': zhTw,
};

const messages: LocalesTypes = LOCALES;

const resolveLanguage = (lng: string): LocalesType => {
    const l = lng.toLowerCase();

    if (messages[l as LocalesType]) {
        return l as LocalesType;
    }

    // Chinese locales
    if (l.startsWith('zh')) {
        if (l.includes('tw') || l.includes('hk') || l.includes('mo')) {
            return 'zh-tw' as LocalesType;
        }
        return 'zh-cn' as LocalesType;
    }

    // Portuguese locales
    if (l.startsWith('pt')) {
        if (l.includes('br')) {
            return 'pt-br' as LocalesType;
        }
        return 'pt-pt' as LocalesType;
    }

    // Try base language (e.g., en-us -> en)
    const base = l.split('-')[0] as LocalesType;
    if (messages[base]) {
        return base;
    }

    return BASE_LOCALE as LocalesType;
};

export const i18n = (lang: LocalesType) => {
    const resolved = resolveLanguage(lang);
    return {
        getMessage: (key: string) => messages[resolved]?.[key] || '',
        getUILanguage: () => resolved,
        getBaseMessage: (key: string) => messages.en![key] || key,
        getBaseUILanguage: () => BASE_LOCALE as LocalesType,
    };
};

const detectedLanguage = ((typeof window !== 'undefined' &&
    typeof localStorage !== 'undefined' &&
    LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.LANGUAGE)) ||
    (typeof navigator !== 'undefined' && (navigator.language as string)) ||
    BASE_LOCALE) as LocalesType;

let currentLanguage: LocalesType = resolveLanguage(detectedLanguage);

let translator = translate.createReactTranslator<any>(i18n(currentLanguage), React, {
    tags: [
        {
            key: 'br',
            createdTag: 'br',
        },
        {
            key: 'strong',
            createdTag: 'strong',
        },
        {
            key: 'code',
            createdTag: 'code',
        },
    ],
});

const intl = {
    getMessage: (key: string, values?: any) => {
        return translator.getMessage(key, values);
    },

    getPlural: (key: string, values?: any) => {
        return translator.getPlural(key, values);
    },

    getUILanguage: () => currentLanguage,

    getBaseMessage: (key: string) => {
        return messages.en?.[key] || key;
    },

    getBaseUILanguage: () => BASE_LOCALE as LocalesType,

    changeLanguage: (newLang: LocalesType) => {
        const resolved = resolveLanguage(newLang);
        currentLanguage = resolved;
        translator = translate.createReactTranslator<any>(i18n(resolved), React, {
            tags: [
                {
                    key: 'br',
                    createdTag: 'br',
                },
                {
                    key: 'strong',
                    createdTag: 'strong',
                },
                {
                    key: 'code',
                    createdTag: 'code',
                },
            ],
        });
    },

    get translator() {
        return translator;
    },
};

export default intl;
