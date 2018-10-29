import React, { Component } from 'react';
import { Trans, withNamespaces } from 'react-i18next';
import { REPOSITORY } from '../../helpers/constants';
import i18n from '../../i18n';

class Footer extends Component {
    getYear = () => {
        const today = new Date();
        return today.getFullYear();
    };

    changeLanguage = () => {
        i18n.changeLanguage(i18n.language === 'en' ? 'vi' : 'en');
    }

    render() {
        return (
            <footer className="footer">
                <div className="container">
                    <div className="row align-items-center flex-row">
                        <div className="col-12 col-lg-auto mt-3 mt-lg-0 text-center">
                            <div className="row align-items-center justify-content-center">
                                <div className="col-auto">
                                    Copyright © {this.getYear()} <a href="https://adguard.com/">AdGuard</a>
                                </div>
                                <div className="col-auto">
                                    <ul className="list-inline text-center mb-0">
                                        <li className="list-inline-item">
                                            <a href={REPOSITORY.URL} target="_blank" rel="noopener noreferrer"><Trans>Homepage</Trans></a>
                                        </li>
                                    </ul>
                                </div>
                                <div className="col-auto">
                                    <button className="btn btn-outline-info btn-sm" onClick={this.changeLanguage}
                                        title={i18n.language === 'en' ? 'Chuyển sang Tiếng Việt' : 'Change to English'}>
                                        {i18n.language === 'en' ? 'English' : 'Tiếng Việt'}
                                    </button>
                                </div>
                                <div className="col-auto">
                                    <a href={`${REPOSITORY.URL}/issues/new`} className="btn btn-outline-primary btn-sm" target="_blank" rel="noopener noreferrer">
                                        <Trans>Report an issue</Trans>
                                    </a>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </footer>
        );
    }
}

export default withNamespaces()(Footer);
