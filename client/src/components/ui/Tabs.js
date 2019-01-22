import React, { Component } from 'react';
import PropTypes from 'prop-types';

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
                children,
            },
            state: {
                activeTab,
            },
        } = this;

        return (
            <div className="tabs">
                <div className="tabs__controls">
                    {children.map((child) => {
                        const { label } = child.props;

                        return (
                            <Tab
                                key={label}
                                label={label}
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
    children: PropTypes.array.isRequired,
};

export default Tabs;
