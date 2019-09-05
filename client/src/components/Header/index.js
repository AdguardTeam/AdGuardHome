import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';

import Menu from './Menu';
import logo from '../ui/svg/logo.svg';
import './Header.css';

class Header extends Component {
    state = {
        isMenuOpen: false,
    };

    toggleMenuOpen = () => {
        this.setState(prevState => ({ isMenuOpen: !prevState.isMenuOpen }));
    };

    closeMenu = () => {
        this.setState({ isMenuOpen: false });
    };

    render() {
        const { dashboard, location } = this.props;
        const { isMenuOpen } = this.state;
        const badgeClass = classnames({
            'badge dns-status': true,
            'badge-success': dashboard.protectionEnabled,
            'badge-danger': !dashboard.protectionEnabled,
        });

        return (
            <div className="header">
                <div className="header__container">
                    <div className="header__row">
                        <div
                            className="header-toggler d-lg-none ml-lg-0 collapsed"
                            onClick={this.toggleMenuOpen}
                        >
                            <span className="header-toggler-icon"></span>
                        </div>
                        <div className="header__column">
                            <div className="d-flex align-items-center">
                                <Link to="/" className="nav-link pl-0 pr-1">
                                    <img src={logo} alt="" className="header-brand-img" />
                                </Link>
                                {!dashboard.proccessing && dashboard.isCoreRunning && (
                                    <span className={badgeClass}>
                                        <Trans>{dashboard.protectionEnabled ? 'on' : 'off'}</Trans>
                                    </span>
                                )}
                            </div>
                        </div>
                        <Menu
                            location={location}
                            isMenuOpen={isMenuOpen}
                            toggleMenuOpen={this.toggleMenuOpen}
                            closeMenu={this.closeMenu}
                        />
                        <div className="header__column">
                            <div className="header__right">
                                <a href="/control/logout" className="btn btn-sm btn-outline-secondary">
                                    <Trans>logout</Trans>
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }
}

Header.propTypes = {
    dashboard: PropTypes.object.isRequired,
    location: PropTypes.object.isRequired,
    getVersion: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Header);
