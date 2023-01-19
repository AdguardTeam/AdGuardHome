import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';

import { REPOSITORY, PRIVACY_POLICY_LINK, THEMES } from '../../helpers/constants';
import { LANGUAGES } from '../../helpers/twosky';
import i18n from '../../i18n';

import Version from './Version';
import './Footer.css';
import './Select.css';
import { setHtmlLangAttr } from '../../helpers/helpers';
import { changeTheme } from '../../actions';

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

    const currentTheme = useSelector((state) => (state.dashboard ? state.dashboard.theme : 'auto'));
    const profileName = useSelector((state) => (state.dashboard ? state.dashboard.name : ''));
    const isLoggedIn = profileName !== '';

    const getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    const changeLanguage = (event) => {
        const { value } = event.target;
        i18n.changeLanguage(value);
        setHtmlLangAttr(value);
    };

    const onThemeChanged = (event) => {
        const { value } = event.target;
        dispatch(changeTheme(value));
    };

    const renderCopyright = () => <div className="footer__column">
        <div className="footer__copyright">
            {t('copyright')} &copy; {getYear()}{' '}
            <a target="_blank" rel="noopener noreferrer" href="https://link.adtidy.org/forward.html?action=home&from=ui&app=home">AdGuard</a>
        </div>
    </div>;

    const renderLinks = (linksData) => linksData.map(({ name, href, className = '' }) => <a
            key={name}
            href={href}
            className={classNames('footer__link', className)}
            target="_blank"
            rel="noopener noreferrer"
        >
            {t(name)}
        </a>);

    const renderThemeSelect = (currentTheme, isLoggedIn) => {
        if (!isLoggedIn) {
            return '';
        }

        return <select
            className="form-control select select--theme"
            value={currentTheme}
            onChange={onThemeChanged}
        >
            {Object.values(THEMES)
                .map((theme) => (
                    <option key={theme} value={theme}>
                        {t(`theme_${theme}`)}
                    </option>
                ))}
        </select>;
    };

    return (
        <>
            <footer className="footer">
                <div className="container">
                    <div className="footer__row">
                        <div className="footer__column footer__column--links">
                            {renderLinks(linksData)}
                        </div>
                        <div className="footer__column footer__column--theme">
                            {renderThemeSelect(currentTheme, isLoggedIn)}
                        </div>
                        <div className="footer__column footer__column--language">
                            <select
                                className="form-control select select--language"
                                value={i18n.language}
                                onChange={changeLanguage}
                            >
                                {Object.keys(LANGUAGES)
                                    .map((lang) => (
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
