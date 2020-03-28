import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';
import { renderTextareaField } from '../../../../helpers/form';

const Form = (props) => {
    const {
        handleSubmit, submitting, invalid, processingSet,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="form__group mb-5">
                <label className="form__label form__label--with-desc" htmlFor="allowed_clients">
                    <Trans>access_allowed_title</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>access_allowed_desc</Trans>
                </div>
                <Field
                    id="allowed_clients"
                    name="allowed_clients"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea"
                    disabled={processingSet}
                />
            </div>
            <div className="form__group mb-5">
                <label className="form__label form__label--with-desc" htmlFor="disallowed_clients">
                    <Trans>access_disallowed_title</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>access_disallowed_desc</Trans>
                </div>
                <Field
                    id="disallowed_clients"
                    name="disallowed_clients"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea"
                    disabled={processingSet}
                />
            </div>
            <div className="form__group mb-5">
                <label className="form__label form__label--with-desc" htmlFor="blocked_hosts">
                    <Trans>access_blocked_title</Trans>
                </label>
                <div className="form__desc form__desc--top">
                    <Trans>access_blocked_desc</Trans>
                </div>
                <Field
                    id="blocked_hosts"
                    name="blocked_hosts"
                    component={renderTextareaField}
                    type="text"
                    className="form-control form-control--textarea"
                    disabled={processingSet}
                />
            </div>
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
};

export default flow([withNamespaces(), reduxForm({ form: 'accessForm' })])(Form);
