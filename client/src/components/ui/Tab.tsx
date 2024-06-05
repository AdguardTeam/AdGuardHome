import React from 'react';
import classnames from 'classnames';
import { useTranslation } from 'react-i18next';

interface TabProps {
    activeTabLabel: string;
    label: string;
    onClick: (...args: unknown[]) => unknown;
    title?: string;
}

const Tab = ({ activeTabLabel, label, title, onClick }: TabProps) => {
    const [t] = useTranslation();
    const handleClick = () => onClick(label);

    const tabClass = classnames({
        tab__control: true,
        'tab__control--active': activeTabLabel === label,
    });

    return (
        <div className={tabClass} onClick={handleClick}>
            <svg className="tab__icon">
                <use xlinkHref={`#${label.toLowerCase()}`} />
            </svg>
            {t(title || label)}
        </div>
    );
};

export default Tab;
