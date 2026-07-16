import { createSignal, createMemo, For, Show } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';
import intl, { LocalesType } from 'panel/common/intl';

import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { LanguageDropdown } from '../LanguageDropdown/LanguageDropdown';
import { REPOSITORY, PRIVACY_POLICY_LINK, THEMES } from 'panel/helpers/constants';
import { LANGUAGES, LANGUAGE_NAMES } from 'panel/helpers/twosky';
import { setHtmlLangAttr, setUITheme } from 'panel/helpers/helpers';
import {
    changeTheme,
    changeLanguage as changeLanguageAction,
    getVersion,
} from 'panel/stores/dashboard';
import { dashboardState } from 'panel/stores/dashboard';

import s from './styles.module.pcss';

export const Footer = () => {
    const currentTheme = () => dashboardState.theme || THEMES.auto;
    const profileName = () => dashboardState.name || '';
    const currentLanguage = () => dashboardState.language || intl.getUILanguage();
    const isLoggedIn = () => profileName() !== '';

    const linksData = createMemo(() => [
        { href: PRIVACY_POLICY_LINK, name: intl.getMessage('privacy_policy') },
        { href: REPOSITORY.ISSUES, name: intl.getMessage('report_an_issue') },
        { href: REPOSITORY.RELEASE_NOTES, name: intl.getMessage('release_notes') },
    ]);

    const themeTranslations = createMemo<Record<string, string>>(() => ({
        auto: intl.getMessage('system_theme'),
        dark: intl.getMessage('dark_theme'),
        light: intl.getMessage('light_theme'),
    }));

    const [currentThemeLocal, setCurrentThemeLocal] = createSignal(THEMES.auto);
    const [themeDropdownOpen, setThemeDropdownOpen] = createSignal(false);

    const getYear = () => new Date().getFullYear();

    const getThemeIcon = () => {
        const activeTheme = isLoggedIn() ? currentTheme() : currentThemeLocal();
        if (activeTheme === THEMES.auto) return 'theme_auto';
        if (activeTheme === THEMES.dark) return 'theme_dark';
        return 'theme_light';
    };

    const changeLanguage = async (newLang: LocalesType) => {
        setHtmlLangAttr(newLang);
        try {
            await changeLanguageAction(newLang);
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, newLang);
            window.location.reload();
        } catch (error) {
            console.error('Failed to save language preference:', error);
        }
    };

    const onThemeChange = (value: string) => {
        if (isLoggedIn()) {
            changeTheme(value);
        } else {
            setUITheme(value);
            setCurrentThemeLocal(value);
        }
        setThemeDropdownOpen(false);
    };

    return (
        <footer class={s.footer}>
            <div class={s.container}>
                <div class={s.leftGroup}>
                    <div class={s.copyright}>&copy; 2018–{getYear()} AdGuard Home</div>

                    <Show when={dashboardState.dnsVersion}>
                        <div class={s.version}>
                            {intl.getMessage('version_number', {
                                value: dashboardState.dnsVersion,
                            })}

                            <Show when={dashboardState.checkUpdateFlag}>
                                <button
                                    type="button"
                                    class={cn(s.checkUpdateBtn, {
                                        [s.checkUpdateBtn_loading]:
                                            dashboardState.processingVersion,
                                    })}
                                    aria-label={intl.getMessage('check_updates_btn')}
                                    disabled={dashboardState.processingVersion}
                                    data-testid="footer-check-updates"
                                    onClick={() => getVersion(true)}
                                >
                                    <Icon
                                        icon={
                                            dashboardState.processingVersion ? 'loader' : 'refresh'
                                        }
                                    />
                                </button>
                            </Show>
                        </div>
                    </Show>

                    <div class={s.links}>
                        <For each={linksData()}>
                            {({ name, href }) => (
                                <a
                                    href={href}
                                    class={cn(theme.link.link, theme.link.noDecoration)}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {name}
                                </a>
                            )}
                        </For>
                    </div>
                </div>

                <div class={s.dropdownWrapper}>
                    <Dropdown
                        open={themeDropdownOpen()}
                        onOpenChange={setThemeDropdownOpen}
                        menu={
                            <div class={theme.dropdown.menu}>
                                <For each={Object.values(THEMES)}>
                                    {(v) => (
                                        <button
                                            type="button"
                                            class={cn(theme.dropdown.item, {
                                                [theme.dropdown.item_active]: currentTheme() === v,
                                            })}
                                            onClick={() => onThemeChange(v)}
                                        >
                                            {themeTranslations()[v]}
                                        </button>
                                    )}
                                </For>
                            </div>
                        }
                        class={s.dropdown}
                        position="bottomRight"
                    >
                        <div class={s.dropdownTrigger}>
                            <Icon icon={getThemeIcon()} class={s.icon} />
                            <span>
                                {
                                    themeTranslations()[
                                        isLoggedIn() ? currentTheme() : currentThemeLocal()
                                    ]
                                }
                            </span>
                        </div>
                    </Dropdown>
                </div>

                <div class={s.dropdownWrapper}>
                    <LanguageDropdown
                        value={currentLanguage()}
                        languages={LANGUAGES}
                        languageNames={LANGUAGE_NAMES}
                        onChange={(lang: string) => changeLanguage(lang as LocalesType)}
                        class={s.dropdown}
                        position="bottomRight"
                    />
                </div>
            </div>
        </footer>
    );
};
