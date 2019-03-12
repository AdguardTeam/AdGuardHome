import React, { Fragment } from 'react';
import { Trans } from 'react-i18next';

import { R_IPV4, UNSAFE_PORTS } from '../helpers/constants';

export const renderField = ({
    input, id, className, placeholder, type, disabled, meta: { touched, error },
}) => (
    <Fragment>
        <input
            {...input}
            id={id}
            placeholder={placeholder}
            type={type}
            className={className}
            disabled={disabled}
        />
        {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

export const renderSelectField = ({
    input, placeholder, disabled, meta: { touched, error },
}) => (
    <Fragment>
        <label className="checkbox checkbox--form">
            <span className="checkbox__marker"/>
            <input
                {...input}
                type="checkbox"
                className="checkbox__input"
                disabled={disabled}
            />
            <span className="checkbox__label">
                <span className="checkbox__label-text checkbox__label-text--long">
                    <span className="checkbox__label-title">{placeholder}</span>
                </span>
            </span>
        </label>
        {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

export const required = (value) => {
    if (value || value === 0) {
        return false;
    }
    return <Trans>form_error_required</Trans>;
};

export const ipv4 = (value) => {
    if (value && !new RegExp(R_IPV4).test(value)) {
        return <Trans>form_error_ip_format</Trans>;
    }
    return false;
};

export const isPositive = (value) => {
    if ((value || value === 0) && (value <= 0)) {
        return <Trans>form_error_positive</Trans>;
    }
    return false;
};

export const port = (value) => {
    if ((value || value === 0) && (value < 80 || value > 65535)) {
        return <Trans>form_error_port_range</Trans>;
    }
    return false;
};

export const isSafePort = (value) => {
    if (UNSAFE_PORTS.includes(value)) {
        return <Trans>form_error_port_unsafe</Trans>;
    }
    return false;
};

export const toNumber = value => value && parseInt(value, 10);
