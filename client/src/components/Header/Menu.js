import React, { Component, Fragment } from 'react';
import { NavLink } from 'react-router-dom';
import PropTypes from 'prop-types';
import enhanceWithClickOutside from 'react-click-outside';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';

import { SETTINGS_URLS } from '../../helpers/constants';
import Dropdown from '../ui/Dropdown';

class Menu extends Component {
    handleClickOutside = () => {
        this.props.closeMenu();
    };

    toggleMenu = () => {
        this.props.toggleMenuOpen();
    };

    getActiveClassForSettings = () => {
        const { pathname } = this.props.location;
        const isSettingsPage = SETTINGS_URLS.some(item => item === pathname);

        return isSettingsPage ? 'active' : '';
    };

    render() {
        const menuClass = classnames({
            'header__column mobile-menu': true,
            'mobile-menu--active': this.props.isMenuOpen,
        });

        const dropdownControlClass = `nav-link ${this.getActiveClassForSettings()}`;

        return (
            <Fragment>
                <div className={menuClass}>
                    <ul className="nav nav-tabs border-0 flex-column flex-lg-row flex-nowrap">
                        <li className="nav-item border-bottom d-lg-none" onClick={this.toggleMenu}>
                            <div className="nav-link nav-link--back">
                                <svg className="nav-icon">
                                    <use xlinkHref="#back" />
                                </svg>
                                <Trans>back</Trans>
                            </div>
                        </li>
                        <li className="nav-item">
                            <NavLink to="/" exact={true} className="nav-link">
                                <svg className="nav-icon">
                                    <use xlinkHref="#dashboard" />
                                </svg>
                                <Trans>dashboard</Trans>
                            </NavLink>
                        </li>
                        <Dropdown
                            label={this.props.t('settings')}
                            baseClassName="dropdown nav-item"
                            controlClassName={dropdownControlClass}
                            icon="settings"
                        >
                            <Fragment>
                                <NavLink to="/settings" className="dropdown-item">
                                    <Trans>general_settings</Trans>
                                </NavLink>
                                <NavLink to="/dns" className="dropdown-item">
                                    <Trans>dns_settings</Trans>
                                </NavLink>
                                <NavLink to="/encryption" className="dropdown-item">
                                    <Trans>encryption_settings</Trans>
                                </NavLink>
                                <NavLink to="/clients" className="dropdown-item">
                                    <Trans>client_settings</Trans>
                                </NavLink>
                                <NavLink to="/dhcp" className="dropdown-item">
                                    <Trans>dhcp_settings</Trans>
                                </NavLink>
                            </Fragment>
                        </Dropdown>
                        <li className="nav-item">
                            <NavLink to="/filters" className="nav-link">
                                <svg className="nav-icon">
                                    <use xlinkHref="#filters" />
                                </svg>
                                <Trans>filters</Trans>
                            </NavLink>
                        </li>
                        <li className="nav-item">
                            <NavLink to="/logs" className="nav-link">
                                <svg className="nav-icon">
                                    <use xlinkHref="#log" />
                                </svg>
                                <Trans>query_log</Trans>
                            </NavLink>
                        </li>
                        <li className="nav-item">
                            <NavLink to="/guide" className="nav-link">
                                <svg className="nav-icon">
                                    <use xlinkHref="#setup" />
                                </svg>
                                <Trans>setup_guide</Trans>
                            </NavLink>
                        </li>
                    </ul>
                </div>
            </Fragment>
        );
    }
}

Menu.propTypes = {
    isMenuOpen: PropTypes.bool,
    closeMenu: PropTypes.func,
    toggleMenuOpen: PropTypes.func,
    location: PropTypes.object,
    t: PropTypes.func,
};

export default withNamespaces()(enhanceWithClickOutside(Menu));
