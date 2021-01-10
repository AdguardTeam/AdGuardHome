import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { useTranslation } from 'react-i18next';

const Tab = ({
    activeTabLabel, label, title, onClick,
}) => {
    const [t] = useTranslation();
    const handleClick = () => onClick(label);

    const tabClass = classnames({
        tab__control: true,
        'tab__control--active': activeTabLabel === label,
    });

    return (
        <div
            className={tabClass}
            onClick={handleClick}
        >
            <svg className="tab__icon">
                <use xlinkHref={`#${label.toLowerCase()}`} />
            </svg>
            {t(title || label)}
        </div>
    );
};

Tab.propTypes = {
    activeTabLabel: PropTypes.string.isRequired,
    label: PropTypes.string.isRequired,
    onClick: PropTypes.func.isRequired,
    title: PropTypes.string,
};

export default Tab;
