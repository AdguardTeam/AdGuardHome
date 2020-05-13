import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';
import classnames from 'classnames';

import Examples from './Examples';
import { renderSelectField } from '../../../../helpers/form';

const getInputFields = (parallel_requests_selected, fastest_addr_selected) => [{
    // eslint-disable-next-line react/display-name
    getTitle: () => <label className="form__label" htmlFor="upstream_dns">
        <Trans>upstream_dns</Trans>
    </label>,
    name: 'upstream_dns',
    type: 'text',
    component: 'textarea',
    className: 'form-control form-control--textarea font-monospace',
    placeholder: 'upstream_dns',
},
{
    name: 'parallel_requests',
    placeholder: 'parallel_requests',
    component: renderSelectField,
    type: 'checkbox',
    subtitle: 'upstream_parallel',
    disabled: fastest_addr_selected,
},
{
    name: 'fastest_addr',
    placeholder: 'fastest_addr',
    component: renderSelectField,
    type: 'checkbox',
    subtitle: 'fastest_addr_desc',
    disabled: parallel_requests_selected,
}];

let Form = (props) => {
    const {
        t,
        handleSubmit,
        testUpstream,
        submitting,
        invalid,
        processingSetConfig,
        processingTestUpstream,
        fastest_addr,
        parallel_requests,
        upstream_dns,
        bootstrap_dns,
    } = props;

    const testButtonClass = classnames({
        'btn btn-primary btn-standard mr-2': true,
        'btn btn-primary btn-standard mr-2 btn-loading': processingTestUpstream,
    });

    const INPUT_FIELDS = getInputFields(parallel_requests, fastest_addr);

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                {INPUT_FIELDS.map(({
                    name, component, type, className, placeholder, getTitle, subtitle, disabled,
                }) => <div className="col-12 mb-4" key={name}>
                    {typeof getTitle === 'function' && getTitle()}
                    <Field
                        id={name}
                        name={name}
                        component={component}
                        type={type}
                        className={className}
                        placeholder={t(placeholder)}
                        subtitle={t(subtitle)}
                        disabled={processingSetConfig || processingTestUpstream || disabled}
                    />
                </div>)}
                <div className="col-12">
                    <Examples />
                    <hr />
                </div>
                <div className="col-12 mb-4">
                    <label
                        className="form__label form__label--with-desc"
                        htmlFor="bootstrap_dns"
                    >
                        <Trans>bootstrap_dns</Trans>
                    </label>
                    <div className="form__desc form__desc--top">
                        <Trans>bootstrap_dns_desc</Trans>
                    </div>
                    <Field
                        id="bootstrap_dns"
                        name="bootstrap_dns"
                        component="textarea"
                        type="text"
                        className="form-control form-control--textarea form-control--textarea-small font-monospace"
                        placeholder={t('bootstrap_dns')}
                        disabled={processingSetConfig}
                    />
                </div>
            </div>
            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="button"
                        className={testButtonClass}
                        onClick={() =>
                            testUpstream({
                                upstream_dns,
                                bootstrap_dns,
                            })
                        }
                        disabled={!upstream_dns || processingTestUpstream}
                    >
                        <Trans>test_upstream_btn</Trans>
                    </button>
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={
                            submitting || invalid || processingSetConfig || processingTestUpstream
                        }
                    >
                        <Trans>apply_btn</Trans>
                    </button>
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    testUpstream: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    initialValues: PropTypes.object,
    upstream_dns: PropTypes.string,
    bootstrap_dns: PropTypes.string,
    fastest_addr: PropTypes.bool,
    parallel_requests: PropTypes.bool,
    processingTestUpstream: PropTypes.bool,
    processingSetConfig: PropTypes.bool,
    t: PropTypes.func,
};

const selector = formValueSelector('upstreamForm');

Form = connect((state) => {
    const upstream_dns = selector(state, 'upstream_dns');
    const bootstrap_dns = selector(state, 'bootstrap_dns');
    const fastest_addr = selector(state, 'fastest_addr');
    const parallel_requests = selector(state, 'parallel_requests');

    return {
        upstream_dns,
        bootstrap_dns,
        fastest_addr,
        parallel_requests,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'upstreamForm',
    }),
])(Form);
