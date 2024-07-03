import React, { Component } from 'react';
import { withTranslation } from 'react-i18next';

import './Checkbox.css';

interface CheckboxProps {
    title: string;
    subtitle: string;
    enabled: boolean;
    handleChange: (...args: unknown[]) => unknown;
    disabled?: boolean;
    t?: (...args: unknown[]) => string;
}

class Checkbox extends Component<CheckboxProps> {
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
                    <span className="checkbox__marker" />

                    <input
                        type="checkbox"
                        className="checkbox__input"
                        onChange={handleChange}
                        checked={enabled}
                        disabled={disabled}
                    />

                    <span className="checkbox__label">
                        <span className="checkbox__label-text">
                            <span className="checkbox__label-title">{t(title)}</span>

                            <span
                                className="checkbox__label-subtitle"
                                dangerouslySetInnerHTML={{ __html: t(subtitle) }}
                            />
                        </span>
                    </span>
                </label>
            </div>
        );
    }
}

export default withTranslation()(Checkbox);
