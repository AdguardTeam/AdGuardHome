import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';
import cn from 'classnames';

import { REPOSITORY, PRIVACY_POLICY_LINK, THEMES } from '../../helpers/constants';
import { LANGUAGES } from '../../helpers/twosky';
import i18n from '../../i18n';

import Version from './Version';
import './Footer.css';
import './Select.css';

import { setHtmlLangAttr, setUITheme } from '../../helpers/helpers';

import { changeTheme } from '../../actions';
import { RootState } from '../../initialState';

const linksData = [
    {
        href: REPOSITORY.URL,
        name: 'homepage',
    },
    {
        href: PRIVACY_POLICY_LINK,
        name: 'privacy_policy',
    },
    {
        href: REPOSITORY.ISSUES,
        className: 'btn btn-outline-primary btn-sm footer__link--report',
        name: 'report_an_issue',
    },
];

const Footer = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const currentTheme = useSelector((state: RootState) => (state.dashboard ? state.dashboard.theme : THEMES.auto));
    const profileName = useSelector((state: RootState) => (state.dashboard ? state.dashboard.name : ''));
    const isLoggedIn = profileName !== '';
    const [currentThemeLocal, setCurrentThemeLocal] = useState(THEMES.auto);

    const getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    const changeLanguage = (event: any) => {
        const { value } = event.target;
        i18n.changeLanguage(value);
        setHtmlLangAttr(value);
    };

    const onThemeChange = (value: any) => {
        if (isLoggedIn) {
            dispatch(changeTheme(value));
        } else {
            setUITheme(value);
            setCurrentThemeLocal(value);
        }
    };

    const renderCopyright = () => (
        <div className="footer__column">
            <div className="footer__copyright">
                {t('copyright')} &copy; {getYear()}{' '}
                <a
                    target="_blank"
                    rel="noopener noreferrer"
                    href="https://link.adtidy.org/forward.html?action=home&from=ui&app=home">
                    AdGuard
                </a>
            </div>
        </div>
    );

    const renderLinks = (linksData: any) =>
        linksData.map(({ name, href, className = '' }: any) => (
            <a
                key={name}
                href={href}
                className={cn('footer__link', className)}
                target="_blank"
                rel="noopener noreferrer">
                {t(name)}
            </a>
        ));

    const renderThemeButtons = () => {
        const currentValue = isLoggedIn ? currentTheme : currentThemeLocal;

        const content = {
            auto: {
                desc: t('theme_auto_desc'),
                icon: '#auto',
            },
            dark: {
                desc: t('theme_dark_desc'),
                icon: '#dark',
            },
            light: {
                desc: t('theme_light_desc'),
                icon: '#light',
            },
        };

        return Object.values(THEMES)

            .map((theme: any) => (
                <button
                    key={theme}
                    type="button"
                    className="btn btn-sm btn-secondary footer__theme-button"
                    onClick={() => onThemeChange(theme)}
                    title={content[theme].desc}>
                    <svg className={cn('footer__theme-icon', { 'footer__theme-icon--active': currentValue === theme })}>
                        <use xlinkHref={content[theme].icon} />
                    </svg>
                </button>
            ));
    };

    return (
        <>
            <footer className="footer">
                <div className="container">
                    <div className="footer__row">
                        <div className="footer__column footer__column--links">{renderLinks(linksData)}</div>

                        <div className="footer__column footer__column--theme">
                            <div className="footer__themes">
                                <div className="btn-group">{renderThemeButtons()}</div>
                            </div>
                        </div>

                        <div className="footer__column footer__column--language">
                            <select
                                className="form-control select select--language"
                                value={i18n.language}
                                onChange={changeLanguage}>
                                {Object.keys(LANGUAGES).map((lang) => (
                                    <option key={lang} value={lang}>
                                        {LANGUAGES[lang]}
                                    </option>
                                ))}
                            </select>
                        </div>
                    </div>
                </div>
            </footer>

            <div className="footer">
                <div className="container">
                    <div className="footer__row">
                        {renderCopyright()}

                        <div className="footer__column footer__column--language">
                            <Version />
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
};

export default Footer;
