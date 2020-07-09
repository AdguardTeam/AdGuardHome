import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { createOnBlurHandler } from './helpers';
import { R_UNIX_ABSOLUTE_PATH, R_WIN_ABSOLUTE_PATH } from './constants';

export const renderField = (props, elementType) => {
    const {
        input, id, className, placeholder, type, disabled, normalizeOnBlur,
        autoComplete, meta: { touched, error }, min, max, step,
    } = props;

    const onBlur = (event) => createOnBlurHandler(event, input, normalizeOnBlur);

    const element = React.createElement(elementType, {
        ...input,
        id,
        className,
        placeholder,
        autoComplete,
        disabled,
        type,
        min,
        max,
        step,
        onBlur,
    });
    return (
        <>
            {element}
            {!disabled && touched && error
            && <span className="form__message form__message--error">{error}</span>}
        </>
    );
};

renderField.propTypes = {
    id: PropTypes.string.isRequired,
    input: PropTypes.object.isRequired,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    disabled: PropTypes.bool,
    autoComplete: PropTypes.bool,
    normalizeOnBlur: PropTypes.func,
    min: PropTypes.number,
    max: PropTypes.number,
    step: PropTypes.number,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

export const renderTextareaField = (props) => renderField(props, 'textarea');

export const renderInputField = (props) => renderField(props, 'input');

export const renderGroupField = ({
    input,
    id,
    className,
    placeholder,
    type,
    disabled,
    autoComplete,
    isActionAvailable,
    removeField,
    meta: { touched, error },
    normalizeOnBlur,
}) => {
    const onBlur = (event) => createOnBlurHandler(event, input, normalizeOnBlur);

    return (
        <Fragment>
            <div className="input-group">
                <input
                    {...input}
                    id={id}
                    placeholder={placeholder}
                    type={type}
                    className={className}
                    disabled={disabled}
                    autoComplete={autoComplete}
                    onBlur={onBlur}
                />
                {isActionAvailable
                && <span className="input-group-append">
                        <button
                            type="button"
                            className="btn btn-secondary btn-icon btn-icon--green"
                            onClick={removeField}
                        >
                            <svg className="icon icon--small">
                                <use xlinkHref="#cross" />
                            </svg>
                        </button>
                    </span>
                }
            </div>
            {!disabled && touched && error
            && <span className="form__message form__message--error">{error}</span>}
        </Fragment>
    );
};

renderGroupField.propTypes = {
    input: PropTypes.object.isRequired,
    id: PropTypes.string,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    disabled: PropTypes.bool,
    autoComplete: PropTypes.bool,
    isActionAvailable: PropTypes.bool,
    removeField: PropTypes.func,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
    normalizeOnBlur: PropTypes.func,
};

export const renderRadioField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    meta: { touched, error },
}) => <Fragment>
    <label className="custom-control custom-radio">
        <input {...input} type="radio" className="custom-control-input" disabled={disabled} />
        <span className="custom-control-label">{placeholder}</span>
        {subtitle && <span
            className="checkbox__label-subtitle"
            dangerouslySetInnerHTML={{ __html: subtitle }}
        />}
    </label>
    {!disabled
    && touched
    && (error && <span className="form__message form__message--error">{error}</span>)}
</Fragment>;

renderRadioField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    subtitle: PropTypes.string,
    disabled: PropTypes.bool,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

export const renderSelectField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    onClick,
    modifier = 'checkbox--form',
    checked,
    meta: { touched, error },
}) => <>
    <label className={`checkbox ${modifier}`} onClick={onClick}>
        <span className="checkbox__marker" />
        <input {...input} type="checkbox" className="checkbox__input" disabled={disabled} checked={input.checked || checked}/>
        <span className="checkbox__label">
                        <span className="checkbox__label-text checkbox__label-text--long">
                            <span className="checkbox__label-title">{placeholder}</span>
                            {subtitle
                            && <span
                                className="checkbox__label-subtitle"
                                dangerouslySetInnerHTML={{ __html: subtitle }}

                            />}
                        </span>
                    </span>
    </label>
    {!disabled
    && touched
    && error && <span className="form__message form__message--error">{error}</span>}
</>;

renderSelectField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    subtitle: PropTypes.string,
    disabled: PropTypes.bool,
    onClick: PropTypes.func,
    modifier: PropTypes.string,
    checked: PropTypes.bool,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

export const renderServiceField = ({
    input,
    placeholder,
    disabled,
    modifier,
    icon,
    meta: { touched, error },
}) => <Fragment>
    <label className={`service custom-switch ${modifier}`}>
        <input
            {...input}
            type="checkbox"
            className="custom-switch-input"
            value={placeholder.toLowerCase()}
            disabled={disabled}
        />
        <span className="service__switch custom-switch-indicator"></span>
        <span className="service__text">{placeholder}</span>
        <svg className="service__icon">
            <use xlinkHref={`#${icon}`} />
        </svg>
    </label>
    {!disabled && touched && error
    && <span className="form__message form__message--error">{error}</span>}
</Fragment>;

renderServiceField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    disabled: PropTypes.bool,
    modifier: PropTypes.string,
    icon: PropTypes.string,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

/**
 * @param value {string}
 * @returns {*|number}
 */
export const toNumber = (value) => value && parseInt(value, 10);

/**
 * @param value {string}
 * @returns {boolean}
 */
export const isValidAbsolutePath = (value) => R_WIN_ABSOLUTE_PATH.test(value)
    || R_UNIX_ABSOLUTE_PATH.test(value);
