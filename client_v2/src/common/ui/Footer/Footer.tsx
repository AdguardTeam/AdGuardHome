import React, { useState, useMemo } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import intl, { LocalesType } from 'panel/common/intl';

import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { REPOSITORY, PRIVACY_POLICY_LINK, THEMES } from '../../../helpers/constants';
import { LANGUAGES } from '../../../helpers/twosky';
import { setHtmlLangAttr, setUITheme } from '../../../helpers/helpers';
import { changeTheme, changeLanguage as changeLanguageAction } from '../../../actions';
import { RootState } from '../../../initialState';

import s from './styles.module.pcss';

const linksData = [
    {
        href: PRIVACY_POLICY_LINK,
        name: intl.getMessage('privacy_policy'),
    },
    {
        href: REPOSITORY.ISSUES,
        name: intl.getMessage('report_an_issue'),
    },
    {
        href: REPOSITORY.RELEASE_NOTES,
        name: intl.getMessage('release_notes'),
    },
];

const themeTranslations = {
    auto: intl.getMessage('system_theme'),
    dark: intl.getMessage('dark_theme'),
    light: intl.getMessage('light_theme'),
};

export const Footer = () => {
    const dispatch = useDispatch();

    const currentTheme = useSelector((state: RootState) => (state.dashboard ? state.dashboard.theme : THEMES.auto));
    const profileName = useSelector((state: RootState) => (state.dashboard ? state.dashboard.name : ''));
    const currentLanguage =
        useSelector((state: RootState) => (state.dashboard ? state.dashboard.language : '')) || intl.getUILanguage();
    const isLoggedIn = profileName !== '';
    const [currentThemeLocal, setCurrentThemeLocal] = useState(THEMES.auto);
    const [themeDropdownOpen, setThemeDropdownOpen] = useState(false);
    const [langDropdownOpen, setLangDropdownOpen] = useState(false);

    const sortedLanguages = useMemo(
        () => Object.keys(LANGUAGES).sort((a, b) => LANGUAGES[a].localeCompare(LANGUAGES[b])),
        [LANGUAGES],
    );

    const getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    const getThemeIcon = (isLoggedIn: boolean, currentTheme: string, currentThemeLocal: string) => {
        const activeTheme = isLoggedIn ? currentTheme : currentThemeLocal;

        if (activeTheme === THEMES.auto) {
            return 'theme_auto';
        }

        if (activeTheme === THEMES.dark) {
            return 'theme_dark';
        }

        return 'theme_light';
    };

    const changeLanguage = async (newLang: LocalesType) => {
        setHtmlLangAttr(newLang);

        try {
            await dispatch(changeLanguageAction(newLang));
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, newLang);
            window.location.reload();
        } catch (error) {
            console.error('Failed to save language preference:', error);
        }
    };

    const onThemeChange = (value: string) => {
        if (isLoggedIn) {
            dispatch(changeTheme(value));
        } else {
            setUITheme(value);
            setCurrentThemeLocal(value);
        }

        setThemeDropdownOpen(false);
    };

    return (
        <footer className={s.footer}>
            <div className={s.container}>
                <div className={s.copyright}>&copy; 2018–{getYear()} AdGuard Home</div>

                <div className={s.links}>
                    {linksData.map(({ name, href }) => (
                        <a
                            key={name}
                            href={href}
                            className={cn(theme.link.link, theme.link.noDecoration)}
                            target="_blank"
                            rel="noopener noreferrer">
                            {name}
                        </a>
                    ))}
                </div>

                <div className={s.column}>
                    <Dropdown
                        trigger="click"
                        open={themeDropdownOpen}
                        onOpenChange={setThemeDropdownOpen}
                        menu={
                            <div className={theme.dropdown.menu}>
                                {Object.values(THEMES).map((v) => (
                                    <button
                                        type="button"
                                        key={v}
                                        className={cn(theme.dropdown.item, {
                                            [theme.dropdown.item_active]: currentTheme === v,
                                        })}
                                        onClick={() => onThemeChange(v)}>
                                        {themeTranslations[v]}
                                    </button>
                                ))}
                            </div>
                        }
                        className={s.dropdown}
                        position="bottomRight">
                        <div className={s.dropdownTrigger}>
                            <Icon icon={getThemeIcon(isLoggedIn, currentTheme, currentThemeLocal)} className={s.icon} />
                            <span>{themeTranslations[isLoggedIn ? currentTheme : currentThemeLocal]}</span>
                        </div>
                    </Dropdown>
                </div>

                <div className={s.column}>
                    <Dropdown
                        trigger="click"
                        open={langDropdownOpen}
                        onOpenChange={setLangDropdownOpen}
                        menu={
                            <div className={cn(theme.dropdown.menu, theme.dropdown.menu_lang)}>
                                {sortedLanguages.map((lang) => (
                                    <button
                                        type="button"
                                        key={lang}
                                        className={cn(theme.dropdown.item, {
                                            [theme.dropdown.item_active]: currentLanguage === lang,
                                        })}
                                        onClick={() => {
                                            changeLanguage(lang as LocalesType);
                                            setLangDropdownOpen(false);
                                        }}>
                                        {LANGUAGES[lang]}
                                    </button>
                                ))}
                            </div>
                        }
                        className={s.dropdown}
                        overlayClassName={s.langOverlay}
                        position="bottomRight">
                        <div className={s.dropdownTrigger}>
                            <Icon icon="lang" className={s.icon} />
                            <span>{LANGUAGES[currentLanguage] || LANGUAGES[intl.getUILanguage()]}</span>
                        </div>
                    </Dropdown>
                </div>
            </div>
        </footer>
    );
};
