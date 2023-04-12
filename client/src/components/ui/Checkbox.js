import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';

import './Checkbox.css';

class Checkbox extends Component {
    render() {
        const {
            title,
            subtitle,
            enabled,
            handleChange,
            disabled,
            t,
        } = this.props;
        return (
            <div className="form__group form__group--checkbox">
                <label className="checkbox checkbox--settings">
                <span className="checkbox__marker"/>
                <input type="checkbox" className="checkbox__input" onChange={handleChange} checked={enabled} disabled={disabled}/>
                <span className="checkbox__label">
                    <span className="checkbox__label-text">
                    <span className="checkbox__label-title">{ t(title) }</span>
                    <span className="checkbox__label-subtitle" dangerouslySetInnerHTML={{ __html: t(subtitle) }}/>
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
    disabled: PropTypes.bool,
    t: PropTypes.func,
};

export default withTranslation()(Checkbox);
