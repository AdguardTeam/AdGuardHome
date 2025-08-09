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

export const i18n = (lang: LocalesType) => ({
    getMessage: (key: string) => messages[lang]?.[key] || '',
    getUILanguage: () => (lang.includes('zh') ? 'zh' : lang),
    getBaseMessage: (key: string) => messages.en![key] || key,
    getBaseUILanguage: () => BASE_LOCALE as LocalesType,
});

let currentLanguage: LocalesType = BASE_LOCALE as LocalesType;

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
        currentLanguage = newLang;
        translator = translate.createReactTranslator<any>(i18n(newLang), React, {
            tags: [
                {
                    key: 'br',
                    createdTag: 'br',
                },
                {
                    key: 'strong',
                    createdTag: 'strong',
                },
            ],
        });
    },

    get translator() {
        return translator;
    },
};

export default intl;
