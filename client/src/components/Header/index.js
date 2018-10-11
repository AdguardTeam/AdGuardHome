import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Menu from './Menu';
import Version from './Version';
import logo from './logo.svg';
import './Header.css';

class Header extends Component {
    state = {
        isMenuOpen: false,
        isDropdownOpen: false,
    };

    toggleMenuOpen = () => {
        this.setState(prevState => ({ isMenuOpen: !prevState.isMenuOpen }));
    };

    closeMenu = () => {
        this.setState({ isMenuOpen: false });
    };

    render() {
        const { dashboard } = this.props;
        const badgeClass = classnames({
            'badge dns-status': true,
            'badge-success': dashboard.protectionEnabled,
            'badge-danger': !dashboard.protectionEnabled,
        });

        return (
            <div className="header">
                <div className="container">
                    <div className="row align-items-center">
                        <div className="header-toggler d-lg-none ml-2 ml-lg-0 collapsed" onClick={this.toggleMenuOpen}>
                            <span className="header-toggler-icon"></span>
                        </div>
                        <div className="col col-lg-3">
                            <div className="d-flex align-items-center">
                                <Link to="/" className="nav-link pl-0 pr-1">
                                    <img src={logo} alt="" className="header-brand-img" />
                                </Link>
                                {!dashboard.proccessing &&
                                    <span className={badgeClass}>
                                        {dashboard.protectionEnabled ? 'ON' : 'OFF'}
                                    </span>
                                }
                            </div>
                        </div>
                        <Menu
                            location={this.props.location}
                            isMenuOpen={this.state.isMenuOpen}
                            toggleMenuOpen={this.toggleMenuOpen}
                            closeMenu={this.closeMenu}
                        />
                        <div className="col col-sm-6 col-lg-3">
                            <Version
                                { ...this.props.dashboard }
                            />
                        </div>
                    </div>
                </div>
            </div>
        );
    }
}

Header.propTypes = {
    dashboard: PropTypes.object,
    location: PropTypes.object,
};

export default Header;
