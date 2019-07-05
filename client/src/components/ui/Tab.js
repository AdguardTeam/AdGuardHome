import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

class Tab extends Component {
    handleClick = () => {
        this.props.onClick(this.props.label);
    }

    render() {
        const {
            activeTab,
            label,
            title,
        } = this.props;

        const tabClass = classnames({
            tab__control: true,
            'tab__control--active': activeTab === label,
        });

        return (
            <div
                className={tabClass}
                onClick={this.handleClick}
            >
                <svg className="tab__icon">
                    <use xlinkHref={`#${label.toLowerCase()}`} />
                </svg>
                {title || label}
            </div>
        );
    }
}

Tab.propTypes = {
    activeTab: PropTypes.string.isRequired,
    label: PropTypes.string.isRequired,
    onClick: PropTypes.func.isRequired,
    title: PropTypes.string,
};

export default Tab;
