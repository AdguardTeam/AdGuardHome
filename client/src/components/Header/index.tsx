import React, { useState } from 'react';

import { Link, useLocation } from 'react-router-dom';
import { shallowEqual, useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import classnames from 'classnames';

import Menu from './Menu';

import { Logo } from '../ui/svg/logo';
import './Header.css';
import { RootState } from '../../initialState';

const Header = () => {
    const [isMenuOpen, setIsMenuOpen] = useState(false);
    const { t } = useTranslation();

    const { protectionEnabled, processing, isCoreRunning, processingProfile, name } = useSelector(
        (state: RootState) => state.dashboard,
        shallowEqual,
    );

    const { pathname } = useLocation();

    const toggleMenuOpen = () => {
        setIsMenuOpen((isMenuOpen) => !isMenuOpen);
    };

    const closeMenu = () => {
        setIsMenuOpen(false);
    };

    const badgeClass = classnames('badge dns-status', {
        'badge-success': protectionEnabled,
        'badge-danger': !protectionEnabled,
    });

    return (
        <div className="header">
            <div className="header__container">
                <div className="header__row">
                    <div className="header-toggler d-lg-none ml-lg-0 collapsed" onClick={toggleMenuOpen}>
                        <span className="header-toggler-icon" />
                    </div>

                    <div className="header__column">
                        <div className="d-flex align-items-center">
                            <Link to="/" className="nav-link pl-0 pr-1">
                                <Logo className="header-brand-img" />
                            </Link>
                            {!processing && isCoreRunning && (
                                <span className={badgeClass}>{t(protectionEnabled ? 'on' : 'off')}</span>
                            )}
                        </div>
                    </div>

                    <Menu pathname={pathname} isMenuOpen={isMenuOpen} closeMenu={closeMenu} />

                    <div className="header__column">
                        <div className="header__right">
                            {!processingProfile && name && (
                                <a href="control/logout" className="btn btn-sm btn-outline-secondary">
                                    {t('sign_out')}
                                </a>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Header;
