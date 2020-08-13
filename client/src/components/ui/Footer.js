import React from 'react';
import { useTranslation } from 'react-i18next';
import classNames from 'classnames';

import { REPOSITORY, PRIVACY_POLICY_LINK } from '../../helpers/constants';
import { LANGUAGES } from '../../helpers/twosky';
import i18n from '../../i18n';

import Version from './Version';
import './Footer.css';
import './Select.css';
import { setHtmlLangAttr } from '../../helpers/helpers';

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

    const getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    const changeLanguage = (event) => {
        const { value } = event.target;
        i18n.changeLanguage(value);
        setHtmlLangAttr(value);
    };

    const renderCopyright = () => <div className="footer__column">
        <div className="footer__copyright">
            {t('copyright')} &copy; {getYear()}{' '}
            <a target="_blank" rel="noopener noreferrer" href="https://adguard.com/">AdGuard</a>
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

    return (
        <>
            <footer className="footer">
                <div className="container">
                    <div className="footer__row">
                        <div className="footer__column footer__column--links">
                            {renderLinks(linksData)}
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
