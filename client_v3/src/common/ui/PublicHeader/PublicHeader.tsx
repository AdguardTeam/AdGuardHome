import { type JSX } from 'solid-js';
import { Logo } from 'panel/common/ui/Sidebar';
import intl, { type LocalesType } from 'panel/common/intl';
import { LanguageDropdown } from 'panel/common/ui/LanguageDropdown/LanguageDropdown';
import { setHtmlLangAttr } from 'panel/helpers/helpers';
import { changeLanguage as changeLanguageAction, dashboardState } from 'panel/stores/dashboard';

import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import { LANGUAGES, LANGUAGE_NAMES } from 'panel/helpers/twosky';
import styles from './PublicHeader.module.pcss';

type Props = {
    dropdownClass?: string;
    dropdownPosition?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft';
    center?: JSX.Element;
    useLocalLanguage?: boolean;
};

export const PublicHeader = (props: Props) => {
    const changeLanguage = async (newLang: LocalesType) => {
        setHtmlLangAttr(newLang);

        if (props.useLocalLanguage) {
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, newLang);
            window.location.reload();
            return;
        }

        try {
            await changeLanguageAction(newLang);
            LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.LANGUAGE, newLang);
            window.location.reload();
        } catch (error) {
            console.error('Failed to save language preference:', error);
        }
    };

    const currentLanguage = () => dashboardState.language || intl.getUILanguage();

    return (
        <div class={styles.header}>
            <div class={styles.headerContent}>
                <div class={styles.logoWrap}>
                    <Logo id="header" />
                </div>
                {props.center}
                <div class={styles.languageWrap}>
                    <LanguageDropdown
                        value={currentLanguage()}
                        languages={LANGUAGES}
                        languageNames={LANGUAGE_NAMES}
                        onChange={changeLanguage}
                        class={props.dropdownClass}
                        position={props.dropdownPosition ?? 'bottomRight'}
                    />
                </div>
            </div>
        </div>
    );
};
