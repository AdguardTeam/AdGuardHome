import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import { createOnBlurHandler } from './helpers';
import { R_MAC_WITHOUT_COLON, R_UNIX_ABSOLUTE_PATH, R_WIN_ABSOLUTE_PATH } from './constants';

export const renderField = (props, elementType) => {
    const {
        input, id, className, placeholder, type, disabled, normalizeOnBlur, onScroll,
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
        onScroll,
    });

    return (
        <>
            {element}
            {!disabled && touched && error
            && <span className="form__message form__message--error"><Trans>{error}</Trans></span>}
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
    onScroll: PropTypes.func,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.string,
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
        <>
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
                            <svg className="icon icon--24">
                                <use xlinkHref="#cross" />
                            </svg>
                        </button>
                    </span>
                }
            </div>
            {!disabled && touched && error
            && <span className="form__message form__message--error"><Trans>{error}</Trans></span>}
        </>
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
        error: PropTypes.string,
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
    && error
    && <span className="form__message form__message--error"><Trans>{error}</Trans></span>}
</Fragment>;

renderRadioField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    subtitle: PropTypes.string,
    disabled: PropTypes.bool,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.string,
    }).isRequired,
};

export const CheckboxField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    onClick,
    modifier = 'checkbox--form',
    meta: { touched, error },
}) => <>
    <label className={`checkbox ${modifier}`} onClick={onClick}>
        <span className="checkbox__marker" />
        <input {...input} type="checkbox" className="checkbox__input" disabled={disabled} />
        <span className="checkbox__label">
            <span className="checkbox__label-text checkbox__label-text--long">
                <span className="checkbox__label-title">{placeholder}</span>
                {subtitle && <span className="checkbox__label-subtitle">{subtitle}</span>}
            </span>
        </span>
    </label>
    {!disabled
    && touched
    && error
    && <span className="form__message form__message--error"><Trans>{error}</Trans></span>}
</>;

CheckboxField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    subtitle: PropTypes.node,
    disabled: PropTypes.bool,
    onClick: PropTypes.func,
    modifier: PropTypes.string,
    checked: PropTypes.bool,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.string,
    }).isRequired,
};

export const renderSelectField = ({
    input,
    meta: { touched, error },
    children,
    label,
}) => {
    const showWarning = touched && error;

    return <>
        {label && <label><Trans>{label}</Trans></label>}
        <select {...input} className='form-control custom-select'>{children}</select>
        {showWarning
        && <span className="form__message form__message--error form__message--left-pad"><Trans>{error}</Trans></span>}
    </>;
};

renderSelectField.propTypes = {
    input: PropTypes.object.isRequired,
    disabled: PropTypes.bool,
    label: PropTypes.string,
    children: PropTypes.oneOfType([PropTypes.array, PropTypes.element]).isRequired,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.string,
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
    && <span className="form__message form__message--error"><Trans>{error}</Trans></span>}
</Fragment>;

renderServiceField.propTypes = {
    input: PropTypes.object.isRequired,
    placeholder: PropTypes.string,
    disabled: PropTypes.bool,
    modifier: PropTypes.string,
    icon: PropTypes.string,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.string,
    }).isRequired,
};

/**
 *
 * @param {string} ip
 * @returns {*}
 */
export const ip4ToInt = (ip) => {
    const intIp = ip.split('.').reduce((int, oct) => (int * 256) + parseInt(oct, 10), 0);
    return Number.isNaN(intIp) ? 0 : intIp;
};

/**
 * @param value {string}
 * @returns {*|number}
 */
export const toNumber = (value) => value && parseInt(value, 10);

/**
 * @param value {string}
 * @returns {*|number}
 */
export const toFloatNumber = (value) => value && parseFloat(value, 10);

/**
 * @param value {string}
 * @returns {boolean}
 */
export const isValidAbsolutePath = (value) => R_WIN_ABSOLUTE_PATH.test(value)
    || R_UNIX_ABSOLUTE_PATH.test(value);

/**
 * @param value {string}
 * @returns {*|string}
 */
export const normalizeMac = (value) => {
    if (value && R_MAC_WITHOUT_COLON.test(value)) {
        return value.match(/.{2}/g).join(':');
    }

    return value;
};
