import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import classNames from 'classnames';

import { REPOSITORY, PRIVACY_POLICY_LINK } from '../../helpers/constants';
import { LANGUAGES } from '../../helpers/twosky';
import i18n from '../../i18n';

import Version from './Version';
import './Footer.css';
import './Select.css';

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

class Footer extends Component {
    getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    changeLanguage = (event) => {
        i18n.changeLanguage(event.target.value);
    };

    renderCopyright = () => <div className="footer__column">
        <div className="footer__copyright">
            <Trans>copyright</Trans> &copy; {this.getYear()}{' '}
            <a target="_blank" rel="noopener noreferrer" href="https://adguard.com/">AdGuard</a>
        </div>
    </div>;

    renderLinks = (linksData) => linksData.map(({ name, href, className = '' }) => <a
        key={name}
        href={href}
        className={classNames('footer__link', className)}
        target="_blank"
        rel="noopener noreferrer"
    >
        <Trans>{name}</Trans>
    </a>);


    render() {
        const {
            dnsVersion, processingVersion, getVersion, checkUpdateFlag,
        } = this.props;

        return (
            <Fragment>
                <footer className="footer">
                    <div className="container">
                        <div className="footer__row">
                            <div className="footer__column footer__column--links">
                                {this.renderLinks(linksData)}
                            </div>
                            <div className="footer__column footer__column--language">
                                <select
                                    className="form-control select select--language"
                                    value={i18n.language}
                                    onChange={this.changeLanguage}
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
                            {this.renderCopyright()}
                            <div className="footer__column footer__column--language">
                                <Version
                                    dnsVersion={dnsVersion}
                                    processingVersion={processingVersion}
                                    getVersion={getVersion}
                                    checkUpdateFlag={checkUpdateFlag}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </Fragment>
        );
    }
}

Footer.propTypes = {
    dnsVersion: PropTypes.string,
    processingVersion: PropTypes.bool,
    getVersion: PropTypes.func,
    checkUpdateFlag: PropTypes.bool,
};

export default withTranslation()(Footer);
