import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { renderInputField } from '../../../../helpers/form';
import { validateIpv4, validateMac, validateRequiredValue } from '../../../../helpers/validators';
import { FORM_NAME } from '../../../../helpers/constants';
import { toggleLeaseModal } from '../../../../actions';

const Form = ({
    handleSubmit,
    reset,
    pristine,
    submitting,
    processingAdding,
}) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const onClick = () => {
        reset();
        dispatch(toggleLeaseModal());
    };

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group">
                    <Field
                        id="mac"
                        name="mac"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_enter_mac')}
                        validate={[validateRequiredValue, validateMac]}
                    />
                </div>
                <div className="form__group">
                    <Field
                        id="ip"
                        name="ip"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_enter_ip')}
                        validate={[validateRequiredValue, validateIpv4]}
                    />
                </div>
                <div className="form__group">
                    <Field
                        id="hostname"
                        name="hostname"
                        component={renderInputField}
                        type="text"
                        className="form-control"
                        placeholder={t('form_enter_hostname')}
                    />
                </div>
            </div>

            <div className="modal-footer">
                <div className="btn-list">
                    <button
                        type="button"
                        className="btn btn-secondary btn-standard"
                        disabled={submitting}
                        onClick={onClick}
                    >
                        <Trans>cancel_btn</Trans>
                    </button>
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={submitting || pristine || processingAdding}
                    >
                        <Trans>save_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    pristine: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    reset: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    processingAdding: PropTypes.bool.isRequired,
};

export default reduxForm({ form: FORM_NAME.LEASE })(Form);
