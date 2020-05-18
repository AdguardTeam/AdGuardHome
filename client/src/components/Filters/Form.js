import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderInputField, required, isValidPath } from '../../helpers/form';

const Form = (props) => {
    const {
        t,
        closeModal,
        handleSubmit,
        processingAddFilter,
        processingConfigFilter,
        whitelist,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="modal-body">
                <div className="form__group">
                    <Field
                        id="name"
                        name="name"
                        type="text"
                        component={renderInputField}
                        className="form-control"
                        placeholder={t('enter_name_hint')}
                        validate={[required]}
                        normalizeOnBlur={data => data.trim()}
                    />
                </div>
                <div className="form__group">
                    <Field
                        id="url"
                        name="url"
                        type="text"
                        component={renderInputField}
                        className="form-control"
                        placeholder={t('enter_url_or_path_hint')}
                        validate={[required, isValidPath]}
                        normalizeOnBlur={data => data.trim()}
                    />
                </div>
                <div className="form__description">
                    {whitelist ? (
                        <Trans>enter_valid_allowlist</Trans>
                    ) : (
                        <Trans>enter_valid_blocklist</Trans>
                    )}
                </div>
            </div>
            <div className="modal-footer">
                <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={closeModal}
                >
                    <Trans>cancel_btn</Trans>
                </button>
                <button
                    type="submit"
                    className="btn btn-success"
                    disabled={processingAddFilter || processingConfigFilter}
                >
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    t: PropTypes.func.isRequired,
    closeModal: PropTypes.func.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    processingAddFilter: PropTypes.bool.isRequired,
    processingConfigFilter: PropTypes.bool.isRequired,
    whitelist: PropTypes.bool,
};

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'filterForm',
    }),
])(Form);
