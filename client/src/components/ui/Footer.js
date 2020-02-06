import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import { LANGUAGES } from '../../helpers/twosky';
import i18n from '../../i18n';

import Version from './Version';
import './Footer.css';
import './Select.css';

class Footer extends Component {
    getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    changeLanguage = (event) => {
        i18n.changeLanguage(event.target.value);
    };

    render() {
        const {
            dnsVersion, processingVersion, getVersion,
        } = this.props;

        return (
            <Fragment>
                <footer className="footer">
                    <div className="container">
                        <div className="footer__row">
                            {!dnsVersion && (
                                <div className="footer__column">
                                    <div className="footer__copyright">
                                        <Trans>copyright</Trans> &copy; {this.getYear()}{' '}
                                        <a target="_blank" rel="noopener noreferrer" href="https://adguard.com/">AdGuard</a>
                                    </div>
                                </div>
                            )}
                            {!dnsVersion && (
                                <div className="footer__column footer__column--language">
                                    <select
                                        className="form-control select select--language"
                                        value={i18n.language}
                                        onChange={this.changeLanguage}
                                    >
                                        {Object.keys(LANGUAGES).map(lang => (
                                            <option key={lang} value={lang}>
                                                {LANGUAGES[lang]}
                                            </option>
                                        ))}
                                    </select>
                                </div>
                            )}
                        </div>
                    </div>
                </footer>
                {dnsVersion && (
                    <div className="footer">
                        <div className="container">
                            <div className="footer__row">
                                <div className="footer__column">
                                    <div className="footer__copyright">
                                        <Trans>copyright</Trans> &copy; {this.getYear()}{' '}
                                        <a target="_blank" rel="noopener noreferrer" href="https://adguard.com/">AdGuard</a>
                                    </div>
                                </div>
                                <div className="footer__column">
                                    <select
                                        className="form-control select select--language"
                                        value={i18n.language}
                                        onChange={this.changeLanguage}
                                    >
                                        {Object.keys(LANGUAGES).map(lang => (
                                            <option key={lang} value={lang}>
                                                {LANGUAGES[lang]}
                                            </option>
                                        ))}
                                    </select>
                                </div>
                                <div className="footer__column footer__column--language">
                                    <Version
                                        dnsVersion={dnsVersion}
                                        processingVersion={processingVersion}
                                        getVersion={getVersion}
                                    />
                                </div>
                            </div>
                        </div>
                    </div>
                )}
            </Fragment>
        );
    }
}

Footer.propTypes = {
    dnsVersion: PropTypes.string,
    processingVersion: PropTypes.bool,
    getVersion: PropTypes.func,
};

export default withNamespaces()(Footer);
