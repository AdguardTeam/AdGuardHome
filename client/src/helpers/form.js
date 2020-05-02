import React, { Fragment } from 'react';
import { Trans } from 'react-i18next';
import PropTypes from 'prop-types';
import {
    R_IPV4, R_MAC, R_HOST, R_IPV6, R_CIDR, R_CIDR_IPV6,
    UNSAFE_PORTS, R_URL_REQUIRES_PROTOCOL, R_WIN_ABSOLUTE_PATH, R_UNIX_ABSOLUTE_PATH,
} from '../helpers/constants';
import { createOnBlurHandler } from './helpers';

export const renderField = (props, elementType) => {
    const {
        input, id, className, placeholder, type, disabled, normalizeOnBlur,
        autoComplete, meta: { touched, error },
    } = props;

    const onBlur = event => createOnBlurHandler(event, input, normalizeOnBlur);

    const element = React.createElement(elementType, {
        ...input,
        id,
        className,
        placeholder,
        autoComplete,
        disabled,
        type,
        onBlur,
    });
    return (
        <Fragment>
            {element}
            {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
        </Fragment>
    );
};

renderField.propTypes = {
    id: PropTypes.string.isRequired,
    input: PropTypes.object.isRequired,
    meta: PropTypes.object.isRequired,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    disabled: PropTypes.bool,
    autoComplete: PropTypes.bool,
    normalizeOnBlur: PropTypes.func,
};

export const renderTextareaField = props => renderField(props, 'textarea');

export const renderInputField = props => renderField(props, 'input');

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
    const onBlur = event => createOnBlurHandler(event, input, normalizeOnBlur);

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
                {isActionAvailable &&
                <span className="input-group-append">
                        <button
                            type="button"
                            className="btn btn-secondary btn-icon"
                            onClick={removeField}
                        >
                            <svg className="icon icon--close">
                                <use xlinkHref="#cross" />
                            </svg>
                        </button>
                    </span>
                }
            </div>
            {!disabled &&
            touched &&
            (error && <span className="form__message form__message--error">{error}</span>)}
        </Fragment>
    );
};

export const renderRadioField = ({
    input, placeholder, disabled, meta: { touched, error },
}) => (
    <Fragment>
        <label className="custom-control custom-radio custom-control-inline">
            <input {...input} type="radio" className="custom-control-input" disabled={disabled} />
            <span className="custom-control-label">{placeholder}</span>
        </label>
        {!disabled &&
        touched &&
        (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

export const renderSelectField = ({
    input,
    placeholder,
    subtitle,
    disabled,
    onClick,
    modifier = 'checkbox--form',
    meta: { touched, error },
}) => (
        <Fragment>
            <label className={`checkbox ${modifier}`} onClick={onClick}>
                <span className="checkbox__marker" />
                <input {...input} type="checkbox" className="checkbox__input" disabled={disabled} />
                <span className="checkbox__label">
                    <span className="checkbox__label-text checkbox__label-text--long">
                        <span className="checkbox__label-title">{placeholder}</span>
                        {subtitle && (
                            <span
                                className="checkbox__label-subtitle"
                                dangerouslySetInnerHTML={{ __html: subtitle }}
                            />
                        )}
                    </span>
                </span>
            </label>
            {!disabled &&
            touched &&
            (error && <span className="form__message form__message--error">{error}</span>)}
        </Fragment>
);

export const renderServiceField = ({
    input,
    placeholder,
    disabled,
    modifier,
    icon,
    meta: { touched, error },
}) => (
    <Fragment>
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
        {!disabled &&
        touched &&
        (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

// Validation functions
// If the value is valid, the validation function should return undefined.
// https://redux-form.com/6.6.3/examples/fieldlevelvalidation/
export const required = (value) => {
    const formattedValue = typeof value === 'string' ? value.trim() : value;
    if (formattedValue || formattedValue === 0 || (formattedValue && formattedValue.length !== 0)) {
        return undefined;
    }
    return <Trans>form_error_required</Trans>;
};

export const ipv4 = (value) => {
    if (value && !R_IPV4.test(value)) {
        return <Trans>form_error_ip4_format</Trans>;
    }
    return undefined;
};

export const clientId = (value) => {
    if (!value) {
        return undefined;
    }
    const formattedValue = value ? value.trim() : value;
    if (formattedValue && !(
        R_IPV4.test(formattedValue)
        || R_IPV6.test(formattedValue)
        || R_MAC.test(formattedValue)
        || R_CIDR.test(formattedValue)
        || R_CIDR_IPV6.test(formattedValue)
    )) {
        return <Trans>form_error_client_id_format</Trans>;
    }
    return undefined;
};

export const ipv6 = (value) => {
    if (value && !R_IPV6.test(value)) {
        return <Trans>form_error_ip6_format</Trans>;
    }
    return undefined;
};

export const ip = (value) => {
    if (value && !R_IPV4.test(value) && !R_IPV6.test(value)) {
        return <Trans>form_error_ip_format</Trans>;
    }
    return undefined;
};

export const mac = (value) => {
    if (value && !R_MAC.test(value)) {
        return <Trans>form_error_mac_format</Trans>;
    }
    return undefined;
};

export const isPositive = (value) => {
    if ((value || value === 0) && value <= 0) {
        return <Trans>form_error_positive</Trans>;
    }
    return undefined;
};

export const biggerOrEqualZero = (value) => {
    if (value < 0) {
        return <Trans>form_error_negative</Trans>;
    }
    return false;
};

export const port = (value) => {
    if ((value || value === 0) && (value < 80 || value > 65535)) {
        return <Trans>form_error_port_range</Trans>;
    }
    return undefined;
};

export const validInstallPort = (value) => {
    if (value < 1 || value > 65535) {
        return <Trans>form_error_port</Trans>;
    }
    return undefined;
};

export const portTLS = (value) => {
    if (value === 0) {
        return undefined;
    } else if (value && (value < 80 || value > 65535)) {
        return <Trans>form_error_port_range</Trans>;
    }
    return undefined;
};

export const isSafePort = (value) => {
    if (UNSAFE_PORTS.includes(value)) {
        return <Trans>form_error_port_unsafe</Trans>;
    }
    return undefined;
};

export const domain = (value) => {
    if (value && !R_HOST.test(value)) {
        return <Trans>form_error_domain_format</Trans>;
    }
    return undefined;
};

export const answer = (value) => {
    if (value && (!R_IPV4.test(value) && !R_IPV6.test(value) && !R_HOST.test(value))) {
        return <Trans>form_error_answer_format</Trans>;
    }
    return undefined;
};

export const isValidUrl = (value) => {
    if (value && !R_URL_REQUIRES_PROTOCOL.test(value)) {
        return <Trans>form_error_url_format</Trans>;
    }
    return undefined;
};

export const isValidAbsolutePath = value => R_WIN_ABSOLUTE_PATH.test(value)
    || R_UNIX_ABSOLUTE_PATH.test(value);

export const isValidPath = (value) => {
    if (value && !isValidAbsolutePath(value) && !R_URL_REQUIRES_PROTOCOL.test(value)) {
        return <Trans>form_error_url_or_path_format</Trans>;
    }
    return undefined;
};

export const toNumber = value => value && parseInt(value, 10);
