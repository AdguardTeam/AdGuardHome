import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { R_IPV4 } from '../../../helpers/constants';

const required = (value) => {
    if (value) {
        return false;
    }
    return 'Required field';
};

const ipv4 = (value) => {
    if (value && !new RegExp(R_IPV4).test(value)) {
        return 'Invalid IPv4 format';
    }
    return false;
};

const renderField = ({
    input, className, placeholder, type, disabled, meta: { touched, error },
}) => (
    <Fragment>
        <input
            {...input}
            placeholder={placeholder}
            type={type}
            className={className}
            disabled={disabled}
        />
        {!disabled && touched && (error && <span className="form__message form__message--error">{error}</span>)}
    </Fragment>
);

const Form = (props) => {
    const {
        handleSubmit, pristine, submitting, enabled,
    } = props;

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--dhcp">
                        <label>Gateway IP</label>
                        <Field
                            name="gateway_ip"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder="Gateway IP"
                            validate={[ipv4, required]}
                            disabled={!enabled}
                        />
                    </div>
                    <div className="form__group form__group--dhcp">
                        <label>Subnet mask</label>
                        <Field
                            name="subnet_mask"
                            component={renderField}
                            type="text"
                            className="form-control"
                            placeholder="Subnet mask"
                            validate={[ipv4, required]}
                            disabled={!enabled}
                        />
                    </div>
                </div>
                <div className="col-lg-6">
                    <div className="form__group form__group--dhcp">
                        <div className="row">
                            <div className="col-12">
                                <label>Range of IP addresses</label>
                            </div>
                            <div className="col">
                                <Field
                                    name="range_start"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder="Range start"
                                    validate={[ipv4, required]}
                                    disabled={!enabled}
                                />
                            </div>
                            <div className="col">
                                <Field
                                    name="range_end"
                                    component={renderField}
                                    type="text"
                                    className="form-control"
                                    placeholder="Range end"
                                    validate={[ipv4, required]}
                                    disabled={!enabled}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="form__group form__group--dhcp">
                        <label>DHCP lease time (in seconds)</label>
                        <Field
                            name="lease_duration"
                            component={renderField}
                            type="number"
                            className="form-control"
                            placeholder="Lease duration"
                            validate={[required]}
                            disabled={!enabled}
                        />
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standart"
                disabled={pristine || submitting || !enabled}
            >
                Save config
            </button>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    pristine: PropTypes.bool,
    submitting: PropTypes.bool,
    enabled: PropTypes.bool,
};

export default reduxForm({
    form: 'dhcpForm',
})(Form);
