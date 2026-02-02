import React from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { Logo } from 'panel/common/ui/Sidebar';
import { RootState } from 'panel/initialState';
import intl, { LocalesType } from 'panel/common/intl';
import { LanguageDropdown } from 'panel/common/ui/LanguageDropdown/LanguageDropdown';
import { setHtmlLangAttr } from 'panel/helpers/helpers';
import { changeLanguage as changeLanguageAction } from 'panel/actions';

import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from 'panel/helpers/localStorageHelper';
import styles from './PublicHeader.module.pcss';

type Props = {
    languages: Record<string, string>;
    dropdownClassName?: string;
    dropdownPosition?: 'bottomRight' | 'bottomLeft' | 'topRight' | 'topLeft';
    center?: React.ReactNode;
};

export const PublicHeader = ({
    languages,
    dropdownClassName,
    dropdownPosition = 'bottomRight',
    center,
}: Props) => {
    const dispatch = useDispatch();

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

    const currentLanguage = useSelector((state: RootState) => (state.dashboard ? state.dashboard.language : '')) || intl.getUILanguage();

    return (
        <div className={styles.header}>
            <div className={styles.headerContent}>
                <div className={styles.logoWrap}>
                    <Logo id="header" />
                </div>
                {center}
                <div className={styles.languageWrap}>
                    <LanguageDropdown
                        value={currentLanguage}
                        languages={languages}
                        onChange={changeLanguage}
                        className={dropdownClassName}
                        position={dropdownPosition}
                    />
                </div>
            </div>
        </div>
    );
};
