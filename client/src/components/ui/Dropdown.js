import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { withTranslation } from 'react-i18next';
import enhanceWithClickOutside from 'react-click-outside';

import './Dropdown.css';

class Dropdown extends Component {
    state = {
        isOpen: false,
    };

    componentDidMount() {
        document.addEventListener('keydown', this.keydown, false);
    }

    componentWillUnmount() {
        document.removeEventListener('keydown', this.keydown, false);
    }

    toggleDropdown = () => {
        this.setState((prevState) => ({ isOpen: !prevState.isOpen }));
    };

    hideDropdown = () => {
        this.setState({ isOpen: false });
    };

    handleClickOutside = () => {
        if (this.state.isOpen) {
            this.hideDropdown();
        }
    };

    keydown = (event) => {
        if (event.key === 'Escape' && this.state.isOpen) {
            this.hideDropdown();
            document.querySelector(`#${this.props.label}-dropdown-button`).focus();
        }
    }

    render() {
        const {
            label,
            controlClassName,
            menuClassName,
            baseClassName,
            icon,
            children,
        } = this.props;

        const { isOpen } = this.state;

        const dropdownClass = classnames({
            [baseClassName]: true,
            show: isOpen,
        });

        const dropdownMenuClass = classnames({
            [menuClassName]: true,
            show: isOpen,
        });

        const ariaSettings = isOpen ? 'true' : 'false';

        return (
            <div className={dropdownClass}>
                <button
                    className={controlClassName}
                    aria-expanded={ariaSettings}
                    onClick={this.toggleDropdown}
                    id={`${label}-dropdown-button`}
                >
                    {icon && (
                        <svg className="nav-icon">
                            <use xlinkHref={`#${icon}`} />
                        </svg>
                    )}
                    {label}
                </button>
                <div className={dropdownMenuClass} onClick={this.hideDropdown}>
                    {children}
                </div>
            </div>
        );
    }
}

Dropdown.defaultProps = {
    baseClassName: 'dropdown',
    menuClassName: 'dropdown-menu dropdown-menu-arrow',
    controlClassName: '',
};

Dropdown.propTypes = {
    label: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
    controlClassName: PropTypes.node.isRequired,
    menuClassName: PropTypes.string.isRequired,
    baseClassName: PropTypes.string.isRequired,
    icon: PropTypes.string,
};

export default withTranslation()(enhanceWithClickOutside(Dropdown));
