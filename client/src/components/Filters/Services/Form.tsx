import React from 'react';

import { Field, reduxForm } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import { toggleAllServices } from '../../../helpers/helpers';

import { renderServiceField } from '../../../helpers/form';
import { FORM_NAME } from '../../../helpers/constants';

interface FormProps {
    blockedServices: unknown[];
    pristine: boolean;
    handleSubmit: (...args: unknown[]) => string;
    change: (...args: unknown[]) => unknown;
    submitting: boolean;
    processing: boolean;
    processingSet: boolean;
    t: (...args: unknown[]) => string;
}

const Form = (props: FormProps) => {
    const { blockedServices, handleSubmit, change, pristine, submitting, processing, processingSet } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="form__group">
                <div className="row mb-4">
                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => toggleAllServices(blockedServices, change, true)}>
                            <Trans>block_all</Trans>
                        </button>
                    </div>

                    <div className="col-6">
                        <button
                            type="button"
                            className="btn btn-secondary btn-block"
                            disabled={processing || processingSet}
                            onClick={() => toggleAllServices(blockedServices, change, false)}>
                            <Trans>unblock_all</Trans>
                        </button>
                    </div>
                </div>

                <div className="services">
                    {blockedServices.map((service: any) => (
                        <Field
                            key={service.id}
                            icon={service.icon_svg}
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
                    disabled={submitting || pristine || processing || processingSet}>
                    <Trans>save_btn</Trans>
                </button>
            </div>
        </form>
    );
};

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.SERVICES,
        enableReinitialize: true,
    }),
])(Form);
