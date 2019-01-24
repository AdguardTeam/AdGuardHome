import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import { renderField, required, ipv4, isPositive, toNumber } from '../../../helpers/form';

const Form = (props) => {
    const {
        t,
        handleSubmit,
        submitting,
        invalid,
        processingConfig,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_gateway_input')}</label>
                        <Field
                            name="gateway_ip"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('dhcp_form_gateway_input')}
                            validate={[ipv4, required]}
                        />
                    </div>
                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_subnet_input')}</label>
                        <Field
                            name="subnet_mask"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder={t('dhcp_form_subnet_input')}
                            validate={[ipv4, required]}
                        />
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>
                            <div className="col">
                                <Field
                                    name="range_start"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder={t('dhcp_form_range_start')}
                                    validate={[ipv4, required]}
                                />
                            </div>
                            <div className="col">
                                <Field
                                    name="range_end"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder={t('dhcp_form_range_end')}
                                    validate={[ipv4, required]}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_lease_title')}</label>
                        <Field
                            name="lease_duration"
                            component={renderField}
                            type="number"
                            className="form-control"
                            placeholder={t('dhcp_form_lease_input')}
                            validate={[required, isPositive]}
                            normalize={toNumber}
                        />
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard"
                disabled={submitting || invalid || processingConfig}
            >
                {t('save_config')}
            </button>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    interfaces: PropTypes.object,
    initialValues: PropTypes.object,
    processingConfig: PropTypes.bool,
    t: PropTypes.func,
};

export default flow([
    withNamespaces(),
    reduxForm({ form: 'dhcpForm' }),
])(Form);
