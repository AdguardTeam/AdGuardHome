import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import { renderTextareaField } from '../../../../helpers/form';
import {
    trimMultilineString,
    removeEmptyLines,
} from '../../../../helpers/helpers';
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

let Form = (props) => {
    const {
        allowedClients, handleSubmit, submitting, invalid, processingSet,
    } = props;

    const renderField = ({
        id, title, subtitle, disabled = false, processingSet, normalizeOnBlur,
    }) => <div key={id} className="form__group mb-5">
        <label className="form__label form__label--with-desc" htmlFor={id}>
            <Trans>{title}</Trans>
            {disabled && <>
                <span> </span>
                (<Trans>disabled</Trans>)
            </>}
        </label>
        <div className="form__desc form__desc--top">
            <Trans components={{ a: <a href={CLIENT_ID_LINK} target="_blank" rel="noopener noreferrer">text</a> }}>{subtitle}</Trans>
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
    </div>;

    renderField.propTypes = {
        id: PropTypes.string,
        title: PropTypes.string,
        subtitle: PropTypes.string,
        disabled: PropTypes.bool,
        normalizeOnBlur: PropTypes.func,
    };

    return (
        <form onSubmit={handleSubmit}>
            {
                fields.map((f) => {
                    const props = { ...f };
                    if (allowedClients && f.id === 'disallowed_clients') {
                        props.disabled = true;
                    }
                    return renderField(props);
                })
            }
            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || invalid || processingSet}
                    >
                        <Trans>save_config</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    initialValues: PropTypes.object.isRequired,
    processingSet: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
    textarea: PropTypes.bool,
    allowedClients: PropTypes.string,
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
