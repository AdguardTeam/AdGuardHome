import React, { Component, Fragment } from 'react';
import { NavLink } from 'react-router-dom';
import PropTypes from 'prop-types';
import enhanceWithClickOutside from 'react-click-outside';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';

import { SETTINGS_URLS } from '../../helpers/constants';
import Dropdown from '../ui/Dropdown';

const MENU_ITEMS = [
    {
        route: '', exact: true, xlinkHref: 'dashboard', text: 'dashboard', order: 0,
    },
    {
        route: 'filters', xlinkHref: 'filters', text: 'filters', order: 2,
    },
    {
        route: 'logs', xlinkHref: 'log', text: 'query_log', order: 3,
    },
    {
        route: 'guide', xlinkHref: 'setup', text: 'setup_guide', order: 4,
    },
];

const DROPDOWN_ITEMS = [
    { route: 'settings', text: 'general_settings' },
    { route: 'dns', text: 'dns_settings' },
    { route: 'encryption', text: 'encryption_settings' },
    { route: 'clients', text: 'client_settings' },
    { route: 'dhcp', text: 'dhcp_settings' },
];

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
                        {MENU_ITEMS.map(({
                            route, text, exact, xlinkHref, order,
                        }) => (
                            <li className={`nav-item order-${order}`} key={text} onClick={this.toggleMenu}>
                                <NavLink to={`/${route}`} exact={exact || false} className="nav-link">
                                    <svg className="nav-icon">
                                        <use xlinkHref={`#${xlinkHref}`} />
                                    </svg>
                                    <Trans>{text}</Trans>
                                </NavLink>
                            </li>
                        ))}
                        <Dropdown
                            label={this.props.t('settings')}
                            baseClassName="dropdown nav-item order-1"
                            controlClassName={dropdownControlClass}
                            icon="settings"
                        >
                            {DROPDOWN_ITEMS.map(({ route, text }) => (
                                <NavLink to={`/${route}`} className="dropdown-item" key={text}
                                         onClick={this.toggleMenu}>
                                    <Trans>{text}</Trans>
                                </NavLink>
                            ))}
                        </Dropdown>
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
