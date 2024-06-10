import React from 'react';
import { connect } from 'react-redux';

import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { renderTextareaField } from '../../../../helpers/form';
import { trimMultilineString, removeEmptyLines } from '../../../../helpers/helpers';
import { CLIENT_ID_LINK, FORM_NAME } from '../../../../helpers/constants';

const fields = [
    {
        id: 'allowed_clients',
        title: 'access_allowed_title',
        subtitle: 'access_allowed_desc',
        normalizeOnBlur: removeEmptyLines,
    },
    {
        id: 'disallowed_clients',
        title: 'access_disallowed_title',
        subtitle: 'access_disallowed_desc',
        normalizeOnBlur: trimMultilineString,
    },
    {
        id: 'blocked_hosts',
        title: 'access_blocked_title',
        subtitle: 'access_blocked_desc',
        normalizeOnBlur: removeEmptyLines,
    },
];

interface FormProps {
    handleSubmit: (...args: unknown[]) => string;
    submitting: boolean;
    invalid: boolean;
    initialValues: object;
    processingSet: boolean;
    t: (...args: unknown[]) => string;
    textarea?: boolean;
    allowedClients?: string;
}

interface renderFieldProps {
    id?: string;
    title?: string;
    subtitle?: string;
    disabled?: boolean;
    processingSet?: boolean;
    normalizeOnBlur?: (...args: unknown[]) => unknown;
}

let Form = (props: FormProps) => {
    const { allowedClients, handleSubmit, submitting, invalid, processingSet } = props;

    const renderField = ({
        id,
        title,
        subtitle,
        disabled = false,
        processingSet,
        normalizeOnBlur,
    }: renderFieldProps) => (
        <div key={id} className="form__group mb-5">
            <label className="form__label form__label--with-desc" htmlFor={id}>
                <Trans>{title}</Trans>

                {disabled && (
                    <>
                        <span> </span>(<Trans>disabled</Trans>)
                    </>
                )}
            </label>

            <div className="form__desc form__desc--top">
                <Trans
                    components={{
                        a: (
                            <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">
                                text
                            </a>
                        ),
                    }}>
                    {subtitle}
                </Trans>
            </div>

            <Field
                id={id}
                name={id}
                component={renderTextareaField}
                type="text"
                className="form-control form-control--textarea font-monospace"
                disabled={disabled || processingSet}
                normalizeOnBlur={normalizeOnBlur}
            />
        </div>
    );

    return (
        <form onSubmit={handleSubmit}>
            {fields.map((f) => {
                return renderField({
                    ...f,
                    disabled: allowedClients && f.id === 'disallowed_clients' || false
                });
            })}

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || invalid || processingSet}>
                        <Trans>save_config</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

const selector = formValueSelector(FORM_NAME.ACCESS);

Form = connect((state) => {
    const allowedClients = selector(state, 'allowed_clients');
    return {
        allowedClients,
    };
})(Form);

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.ACCESS,
    }),
])(Form);
