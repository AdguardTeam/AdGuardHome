import React, { Fragment } from 'react';

const renderField = ({
    input, className, placeholder, type, disabled, autoComplete, meta: { touched, error },
}) => (
    <Fragment>
        <input
            {...input}
            placeholder={placeholder}
            type={type}
            className={className}
            disabled={disabled}
            autoComplete={autoComplete}
        />
        {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

export default renderField;
