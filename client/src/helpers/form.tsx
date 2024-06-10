import React, { Fragment } from 'react';
import { Trans } from 'react-i18next';
import cn from 'classnames';

import { createOnBlurHandler } from './helpers';
import { R_MAC_WITHOUT_COLON, R_UNIX_ABSOLUTE_PATH, R_WIN_ABSOLUTE_PATH } from './constants';

interface renderFieldProps {
    id: string;
    input: object;
    className?: string;
    placeholder?: string;
    type?: string;
    disabled?: boolean;
    autoComplete?: string;
    normalizeOnBlur?: (...args: unknown[]) => unknown;
    min?: number;
    max?: number;
    step?: number;
    onScroll?: (...args: unknown[]) => unknown;
    meta: {
        touched?: boolean;
        error?: string;
    };
}

export const renderField = (props: renderFieldProps, elementType: any) => {
    const {
        input,
        id,
        className,
        placeholder,
        type,
        disabled,
        normalizeOnBlur,
        onScroll,
        autoComplete,
        meta: { touched, error },
        min,
        max,
        step,
    } = props;

    const onBlur = (event: any) => createOnBlurHandler(event, input, normalizeOnBlur);

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
            {!disabled && touched && error && (
                <span className="form__message form__message--error">
                    <Trans>{error}</Trans>
                </span>
            )}
        </>
    );
};

export const renderTextareaField = (props: any) => renderField(props, 'textarea');

export const renderInputField = (props: any) => renderField(props, 'input');

interface renderGroupFieldProps {
    input: object;
    id?: string;
    className?: string;
    placeholder?: string;
    type?: string;
    disabled?: boolean;
    autoComplete?: string;
    isActionAvailable?: boolean;
    removeField?: (...args: unknown[]) => unknown;
    meta: {
        touched?: boolean;
        error?: string;
    };
    normalizeOnBlur?: (...args: unknown[]) => unknown;
}

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
}: renderGroupFieldProps) => {
    const onBlur = (event: any) => createOnBlurHandler(event, input, normalizeOnBlur);

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
                {isActionAvailable && (
                    <span className="input-group-append">
                        <button
                            type="button"
                            className="btn btn-secondary btn-icon btn-icon--green"
                            onClick={removeField}>
                            <svg className="icon icon--24">
                                <use xlinkHref="#cross" />
                            </svg>
                        </button>
                    </span>
                )}
            </div>
            {!disabled && touched && error && (
                <span className="form__message form__message--error">
                    <Trans>{error}</Trans>
                </span>
            )}
        </>
    );
};

interface renderRadioFieldProps {
    input: object;
    placeholder?: string;
    subtitle?: string;
    disabled?: boolean;
    meta: {
        touched?: boolean;
        error?: string;
    };
}

export const renderRadioField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    meta: { touched, error },
}: renderRadioFieldProps) => (
    <Fragment>
        <label className="custom-control custom-radio">
            <input {...input} type="radio" className="custom-control-input" disabled={disabled} />

            <span className="custom-control-label">{placeholder}</span>

            {subtitle && <span className="checkbox__label-subtitle" dangerouslySetInnerHTML={{ __html: subtitle }} />}
        </label>
        {!disabled && touched && error && (
            <span className="form__message form__message--error">
                <Trans>{error}</Trans>
            </span>
        )}
    </Fragment>
);

interface CheckboxFieldProps {
    input: object;
    placeholder?: string;
    subtitle?: React.ReactNode;
    disabled?: boolean;
    onClick?: (...args: unknown[]) => unknown;
    modifier?: string;
    checked?: boolean;
    meta: {
        touched?: boolean;
        error?: string;
    };
}

export const CheckboxField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    onClick,
    modifier = 'checkbox--form',
    meta: { touched, error },
}: CheckboxFieldProps) => (
    <>
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
        {!disabled && touched && error && (
            <div className="form__message form__message--error mt-1">
                <Trans>{error}</Trans>
            </div>
        )}
    </>
);

interface renderSelectFieldProps {
    input: object;
    disabled?: boolean;
    label?: string;
    children: unknown[] | React.ReactElement;
    meta: {
        touched?: boolean;
        error?: string;
    };
}

export const renderSelectField = ({ input, meta: { touched, error }, children, label }: renderSelectFieldProps) => {
    const showWarning = touched && error;

    return (
        <>
            {label && (
                <label>
                    <Trans>{label}</Trans>
                </label>
            )}

            <select {...input} className="form-control custom-select">
                {children}
            </select>
            {showWarning && (
                <span className="form__message form__message--error form__message--left-pad">
                    <Trans>{error}</Trans>
                </span>
            )}
        </>
    );
};

interface renderServiceFieldProps {
    input: object;
    placeholder?: string;
    disabled?: boolean;
    modifier?: string;
    icon?: string;
    meta: {
        touched?: boolean;
        error?: string;
    };
}

export const renderServiceField = ({
    input,
    placeholder,
    disabled,
    modifier,
    icon,
    meta: { touched, error },
}: renderServiceFieldProps) => (
    <>
        <label className={cn('service custom-switch', { [modifier]: modifier })}>
            <input
                {...input}
                type="checkbox"
                className="custom-switch-input"
                value={placeholder.toLowerCase()}
                disabled={disabled}
            />

            <span className="service__switch custom-switch-indicator"></span>

            <span className="service__text" title={placeholder}>
                {placeholder}
            </span>
            {icon && <div dangerouslySetInnerHTML={{ __html: window.atob(icon) }} className="service__icon" />}
        </label>
        {!disabled && touched && error && (
            <span className="form__message form__message--error">
                <Trans>{error}</Trans>
            </span>
        )}
    </>
);

/**
 *
 * @param {string} ip
 * @returns {*}
 */
export const ip4ToInt = (ip: any) => {
    const intIp = ip.split('.').reduce((int: any, oct: any) => int * 256 + parseInt(oct, 10), 0);
    return Number.isNaN(intIp) ? 0 : intIp;
};

/**
 * @param value {string}
 * @returns {*|number}
 */
export const toNumber = (value: any) => value && parseInt(value, 10);

/**
 * @param value {string}
 * @returns {*|number}
 */

export const toFloatNumber = (value: any) => value && parseFloat(value);

/**
 * @param value {string}
 * @returns {boolean}
 */
export const isValidAbsolutePath = (value: any) => R_WIN_ABSOLUTE_PATH.test(value) || R_UNIX_ABSOLUTE_PATH.test(value);

/**
 * @param value {string}
 * @returns {*|string}
 */
export const normalizeMac = (value: any) => {
    if (value && R_MAC_WITHOUT_COLON.test(value)) {
        return value.match(/.{2}/g).join(':');
    }

    return value;
};
