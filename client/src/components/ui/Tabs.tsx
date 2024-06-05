import React from 'react';
import classnames from 'classnames';

import Tab from './Tab';
import './Tabs.css';

interface TabsProps {
    controlClass?: string;
    tabs: object;
    activeTabLabel: string;
    setActiveTabLabel: (...args: unknown[]) => unknown;
    children: React.ReactElement;
}

const Tabs = (props: TabsProps) => {
    const { tabs, controlClass, activeTabLabel, setActiveTabLabel, children: activeTab } = props;

    const onClickTabControl = (tabLabel: any) => setActiveTabLabel(tabLabel);

    const getControlClass = classnames({
        tabs__controls: true,
        [`tabs__controls--${controlClass}`]: controlClass,
    });

    return (
        <div className="tabs">
            <div className={getControlClass}>
                {Object.values(tabs).map((props: any) => {
                    // eslint-disable-next-line react/prop-types
                    const { title, label = title } = props;
                    return (
                        <Tab
                            key={label}
                            label={label}
                            title={title}
                            activeTabLabel={activeTabLabel}
                            onClick={onClickTabControl}
                        />
                    );
                })}
            </div>

            <div className="tabs__content">{activeTab}</div>
        </div>
    );
};

export default Tabs;
