import React, { useState, useRef, useCallback } from 'react';
import classnames from 'classnames';
import { withTranslation } from 'react-i18next';

import { useClickOutside } from '../../hooks/useClickOutside';

import './Dropdown.css';

type DropdownProps = {
    label: string;
    children: React.ReactNode;
    controlClassName: string;
    menuClassName?: string;
    baseClassName?: string;
    icon?: string;
};

const Dropdown = ({
    label,
    controlClassName,
    menuClassName = 'dropdown-menu dropdown-menu-arrow',
    baseClassName = 'dropdown',
    icon,
    children,
}: DropdownProps) => {
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    const toggleDropdown = () => {
        setIsOpen((prev) => !prev);
    };

    const hideDropdown = () => {
        setIsOpen(false);
    };

    const handleClickOutside = useCallback(() => {
        if (isOpen) {
            hideDropdown();
        }
    }, [isOpen]);

    useClickOutside(dropdownRef, handleClickOutside);

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
        <div className={dropdownClass} ref={dropdownRef}>
            <a className={controlClassName} aria-expanded={ariaSettings} onClick={toggleDropdown}>
                {icon && (
                    <svg className="nav-icon">
                        <use xlinkHref={`#${icon}`} />
                    </svg>
                )}
                {label}
            </a>

            <div className={dropdownMenuClass} onClick={hideDropdown}>
                {children}
            </div>
        </div>
    );
};

export default withTranslation()(Dropdown);
