import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { toggleAllServices } from '../../../helpers/helpers';
import { renderServiceField } from '../../../helpers/form';
import { FORM_NAME, SERVICES } from '../../../helpers/constants';

const Form = (props) => {
    const {
        handleSubmit,
        change,
        pristine,
        submitting,
        processing,
        processingSet,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="form__group">
                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => toggleAllServices(SERVICES, change, true)}
                        >
                            <Trans>block_all</Trans>
                        </button>
                    </div>
                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => toggleAllServices(SERVICES, change, false)}
                        >
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>
                <div className="services">
                    {SERVICES.map((service) => (
                        <Field
                            key={service.id}
                            icon={`service_${service.id}`}
                            name={`blocked_services.${service.id}`}
                            type="checkbox"
                            component={renderServiceField}
                            placeholder={service.name}
                            disabled={processing || processingSet}
                        />
                    ))}
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    className="btn btn-success btn-standard btn-large"
                    disabled={submitting || pristine || processing || processingSet}
                >
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};

Form.propTypes = {
    pristine: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    change: PropTypes.func.isRequired,
    submitting: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    processingSet: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.SERVICES,
        enableReinitialize: true,
    }),
])(Form);
