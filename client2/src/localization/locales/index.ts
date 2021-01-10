import React from 'react';
import 'dayjs/locale/ru';

import { PickerLocale } from 'antd/es/date-picker/generatePicker';
import ruPicker from 'antd/es/date-picker/locale/ru_RU';
import enPicker from 'antd/es/date-picker/locale/en_GB';

import ruLang from './ru.json';
import enLang from './en.json';

export enum Locale {
    en = 'en',
    ru = 'ru',
}
export const DatePickerLocale: Record<Locale, PickerLocale> = {
    [Locale.ru]: ruPicker,
    [Locale.en]: enPicker,
};

export const messages: Record<Locale, Record<string, string>> = {
    [Locale.ru]: ruLang,
    [Locale.en]: enLang,
};

export const DEFAULT_LOCALE = Locale.en;

export const reactFormater = (data: (JSX.Element | string)[]) => {
    if (data.every((d) => typeof d === 'string')) {
        return data.join('');
    }
    return React.Children.toArray(data);
};

export const LANGUAGES: { code: Locale; name: string }[] = [
    {
        code: Locale.en,
        name: 'English',
    },
    {
        code: Locale.ru,
        name: 'Русский',
    },
];
