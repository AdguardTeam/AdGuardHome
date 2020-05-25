import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';
import classnames from 'classnames';
import { nanoid } from 'nanoid';

import Examples from './Examples';
import { renderRadioField } from '../../../../helpers/form';
import { DNS_REQUEST_OPTIONS } from '../../../../helpers/constants';

const getInputFields = () => [{
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
    name: 'dnsRequestOption',
    type: 'radio',
    value: DNS_REQUEST_OPTIONS.PARALLEL_REQUESTS,
    component: renderRadioField,
    subtitle: 'upstream_parallel',
    placeholder: 'parallel_requests',
},
{
    name: 'dnsRequestOption',
    type: 'radio',
    value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
    component: renderRadioField,
    subtitle: 'fastest_addr_desc',
    placeholder: 'fastest_addr',
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
        upstream_dns,
        bootstrap_dns,
    } = props;

    const testButtonClass = classnames({
        'btn btn-primary btn-standard mr-2': true,
        'btn btn-primary btn-standard mr-2 btn-loading': processingTestUpstream,
    });

    const INPUT_FIELDS = getInputFields();

    return <form onSubmit={handleSubmit}>
        <div className="row">
            {INPUT_FIELDS.map(({
                name, component, type, className, placeholder, getTitle, subtitle, disabled, value,
            }) => <div className="col-12 mb-4" key={nanoid()}>
                {typeof getTitle === 'function' && getTitle()}
                <Field
                    id={name}
                    value={value}
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
                    onClick={() => testUpstream({
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
    </form>;
};

Form.propTypes = {
    handleSubmit: PropTypes.func,
    testUpstream: PropTypes.func,
    submitting: PropTypes.bool,
    invalid: PropTypes.bool,
    initialValues: PropTypes.object,
    upstream_dns: PropTypes.string,
    bootstrap_dns: PropTypes.string,
    processingTestUpstream: PropTypes.bool,
    processingSetConfig: PropTypes.bool,
    t: PropTypes.func,
};

const selector = formValueSelector('upstreamForm');

Form = connect((state) => {
    const upstream_dns = selector(state, 'upstream_dns');
    const bootstrap_dns = selector(state, 'bootstrap_dns');

    return {
        upstream_dns,
        bootstrap_dns,
    };
})(Form);

export default flow([
    withTranslation(),
    reduxForm({
        form: 'upstreamForm',
    }),
])(Form);
