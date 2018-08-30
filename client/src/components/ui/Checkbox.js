import React, { Component } from 'react';
import PropTypes from 'prop-types';

import './Checkbox.css';

class Checkbox extends Component {
    render() {
        const {
            title,
            subtitle,
            enabled,
            handleChange,
        } = this.props;
        return (
            <div className="form__group">
                <label className="checkbox checkbox--settings">
                <span className="checkbox__marker"/>
                <input type="checkbox" className="checkbox__input" onChange={handleChange} checked={enabled}/>
                <span className="checkbox__label">
                    <span className="checkbox__label-text">
                    <span className="checkbox__label-title">{title}</span>
                    <span className="checkbox__label-subtitle" dangerouslySetInnerHTML={{ __html: subtitle }}/>
                    </span>
                </span>
                </label>
            </div>
        );
    }
}

Checkbox.propTypes = {
    title: PropTypes.string.isRequired,
    subtitle: PropTypes.string.isRequired,
    enabled: PropTypes.bool.isRequired,
    handleChange: PropTypes.func.isRequired,
};

export default Checkbox;
