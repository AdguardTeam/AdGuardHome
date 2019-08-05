import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Tab from './Tab';
import './Tabs.css';

class Tabs extends Component {
    state = {
        activeTab: this.props.children[0].props.label,
    };

    onClickTabControl = (tab) => {
        this.setState({ activeTab: tab });
    }

    render() {
        const {
            props: {
                controlClass,
                children,
            },
            state: {
                activeTab,
            },
        } = this;

        const getControlClass = classnames({
            tabs__controls: true,
            [`tabs__controls--${controlClass}`]: controlClass,
        });

        return (
            <div className="tabs">
                <div className={getControlClass}>
                    {children.map((child) => {
                        const { label, title } = child.props;

                        return (
                            <Tab
                                key={label}
                                label={label}
                                title={title}
                                activeTab={activeTab}
                                onClick={this.onClickTabControl}
                            />
                        );
                    })}
                </div>
                <div className="tabs__content">
                    {children.map((child) => {
                        if (child.props.label !== activeTab) {
                            return false;
                        }
                        return child.props.children;
                    })}
                </div>
            </div>
        );
    }
}

Tabs.propTypes = {
    controlClass: PropTypes.string,
    children: PropTypes.array.isRequired,
};

export default Tabs;
