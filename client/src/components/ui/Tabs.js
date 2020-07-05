import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import Tab from './Tab';
import './Tabs.css';

const Tabs = (props) => {
    const {
        tabs, controlClass, activeTabLabel, setActiveTabLabel, children: activeTab,
    } = props;

    const onClickTabControl = (tabLabel) => setActiveTabLabel(tabLabel);

    const getControlClass = classnames({
        tabs__controls: true,
        [`tabs__controls--${controlClass}`]: controlClass,
    });

    return (
        <div className="tabs">
            <div className={getControlClass}>
                {Object.values(tabs)
                    .map((props) => {
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
            <div className="tabs__content">
                {activeTab}
            </div>
        </div>
    );
};

Tabs.propTypes = {
    controlClass: PropTypes.string,
    tabs: PropTypes.object.isRequired,
    activeTabLabel: PropTypes.string.isRequired,
    setActiveTabLabel: PropTypes.func.isRequired,
    children: PropTypes.element.isRequired,
};

export default Tabs;
