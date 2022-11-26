import React, { Component } from 'react';
import { NavLink } from 'react-router-dom';
import PropTypes from 'prop-types';
import enhanceWithClickOutside from 'react-click-outside';
import classnames from 'classnames';
import { Trans, withTranslation } from 'react-i18next';
import { SETTINGS_URLS, FILTERS_URLS, MENU_URLS } from '../../helpers/constants';
import Dropdown from '../ui/Dropdown';

const DASHBOARD = {
    route: MENU_URLS.root,
    exact: true,
    icon: 'dashboard',
    text: 'dashboard',
};

const MENU_ITEMS = [
    {
        route: MENU_URLS.logs,
        icon: 'log',
        text: 'query_log',
    },
    {
        route: MENU_URLS.guide,
        icon: 'setup',
        text: 'setup_guide',
    },
];

const SETTINGS_ITEMS = [
    {
        route: SETTINGS_URLS.settings,
        text: 'general_settings',
    },
    {
        route: SETTINGS_URLS.dns,
        text: 'dns_settings',
    },
    {
        route: SETTINGS_URLS.encryption,
        text: 'encryption_settings',
    },
    {
        route: SETTINGS_URLS.clients,
        text: 'client_settings',
    },
    {
        route: SETTINGS_URLS.dhcp,
        text: 'dhcp_settings',
    },
];

const FILTERS_ITEMS = [
    {
        route: FILTERS_URLS.dns_blocklists,
        text: 'dns_blocklists',
    },
    {
        route: FILTERS_URLS.dns_allowlists,
        text: 'dns_allowlists',
    },
    {
        route: FILTERS_URLS.dns_rewrites,
        text: 'dns_rewrites',
    },
    {
        route: FILTERS_URLS.blocked_services,
        text: 'blocked_services',
    },
    {
        route: FILTERS_URLS.custom_rules,
        text: 'custom_filtering_rules',
    },
];

class Menu extends Component {
    handleClickOutside = () => {
        this.props.closeMenu();
    };

    closeMenu = () => {
        this.props.closeMenu();
    };

    getActiveClassForDropdown = (URLS) => {
        const isActivePage = Object.values(URLS)
            .some((item) => item === this.props.pathname);

        return isActivePage ? 'active' : '';
    };

    getNavLink = ({
        route, exact, text, className, icon,
    }) => (
        <NavLink
            to={route}
            key={route}
            exact={exact || false}
            className={className}
            onClick={this.closeMenu}
        >
            {icon && (
                <svg className="nav-icon">
                    <use xlinkHref={`#${icon}`} />
                </svg>
            )}
            <Trans>{text}</Trans>
        </NavLink>
    );

    getDropdown = ({
        label, URLS, icon, ITEMS,
    }) => (
        <Dropdown
            label={this.props.t(label)}
            baseClassName='dropdown'
            controlClassName={`nav-link ${this.getActiveClassForDropdown(URLS)}`}
            icon={icon}>
            {ITEMS.map((item) => (
                this.getNavLink({
                    ...item,
                    className: 'dropdown-item',
                })))}
        </Dropdown>
    );

    render() {
        const menuClass = classnames({
            'header__column mobile-menu': true,
            'mobile-menu--active': this.props.isMenuOpen,
        });
        return (
                <div className={menuClass}>
                    <ul className="nav nav-tabs border-0 flex-column flex-lg-row flex-nowrap">
                        <li
                            className={'nav-item'}
                            key={'dashboard'}
                            onClick={this.closeMenu}
                        >
                            {this.getNavLink({
                                ...DASHBOARD,
                                className: 'nav-link',
                            })}
                        </li>
                        <li className="nav-item">
                            {this.getDropdown({
                                label: 'settings',
                                icon: 'settings',
                                URLS: SETTINGS_URLS,
                                ITEMS: SETTINGS_ITEMS,
                            })}
                        </li>
                        <li className="nav-item">
                            {this.getDropdown({
                                label: 'filters',
                                icon: 'filters',
                                URLS: FILTERS_URLS,
                                ITEMS: FILTERS_ITEMS,
                            })}
                        </li>
                        {MENU_ITEMS.map((item) => (
                            <li
                                className={'nav-item'}
                                key={item.text}
                                onClick={this.closeMenu}
                            >
                                {this.getNavLink({
                                    ...item,
                                    className: 'nav-link',
                                })}
                            </li>
                        ))}
                    </ul>
                </div>
        );
    }
}

Menu.propTypes = {
    isMenuOpen: PropTypes.bool.isRequired,
    closeMenu: PropTypes.func.isRequired,
    pathname: PropTypes.string.isRequired,
    t: PropTypes.func,
};

export default withTranslation()(enhanceWithClickOutside(Menu));
