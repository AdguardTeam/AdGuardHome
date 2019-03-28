import React, { Component } from 'react';
import PropTypes from 'prop-types';

import './Accordion.css';

class Accordion extends Component {
    state = {
        isOpen: false,
    }

    handleClick = () => {
        this.setState(prevState => ({ isOpen: !prevState.isOpen }));
    };

    render() {
        const accordionClass = this.state.isOpen
            ? 'accordion__label accordion__label--open'
            : 'accordion__label';

        return (
            <div className="accordion">
                <div
                    className={accordionClass}
                    onClick={this.handleClick}
                >
                    {this.props.label}
                </div>
                {this.state.isOpen && (
                    <div className="accordion__content">
                        {this.props.children}
                    </div>
                )}
            </div>
        );
    }
}

Accordion.propTypes = {
    children: PropTypes.node.isRequired,
    label: PropTypes.string.isRequired,
};

export default Accordion;
