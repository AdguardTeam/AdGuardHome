import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';
import classnames from 'classnames';

import { renderSelectField } from '../../../helpers/form';
import Examples from './Examples';

let Form = (props) => {
    const {
        t,
        handleSubmit,
        testUpstream,
        upstreamDns,
        bootstrapDns,
        allServers,
        submitting,
        invalid,
        processingSetUpstream,
        processingTestUpstream,
    } = props;

    const testButtonClass = classnames({
        'btn btn-primary btn-standard mr-2': true,
        'btn btn-primary btn-standard mr-2 btn-loading': processingTestUpstream,
    });

    return (
        <form onSubmit={handleSubmit}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="upstream_dns">
                            <Trans>upstream_dns</Trans>
                        </label>
                        <Field
                            id="upstream_dns"
                            name="upstream_dns"
                            component="textarea"
                            type="text"
                            className="form-control form-control--textarea"
                            placeholder={t('upstream_dns')}
                        />
                    </div>
                </div>
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Field
                            name="all_servers"
                            type="checkbox"
                            component={renderSelectField}
                            placeholder={t('upstream_parallel')}
                        />
                    </div>
                </div>
                <div className="col-12">
                    <Examples />
                    <hr/>
                </div>
                <div className="col-12">
                    <div className="form__group">
                        <label className="form__label" htmlFor="bootstrap_dns">
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
                            className="form-control"
                            placeholder={t('bootstrap_dns')}
                        />
                    </div>
                </div>
            </div>
            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="button"
                        className={testButtonClass}
                        onClick={() => testUpstream({
                            upstream_dns: upstreamDns,
                            bootstrap_dns: bootstrapDns,
                            all_servers: allServers,
                        })}
                        disabled={!upstreamDns || processingTestUpstream}
                    >
                        <Trans>test_upstream_btn</Trans>
                    </button>
                    <button
                        type="submit"
                        className="btn btn-success btn-standard"
                        disabled={
                            submitting
                            || invalid
                            || processingSetUpstream
                            || processingTestUpstream
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
    upstreamDns: PropTypes.string,
    bootstrapDns: PropTypes.string,
    allServers: PropTypes.bool,
    processingTestUpstream: PropTypes.bool,
    processingSetUpstream: PropTypes.bool,
    t: PropTypes.func,
};

const selector = formValueSelector('upstreamForm');

Form = connect((state) => {
    const upstreamDns = selector(state, 'upstream_dns');
    const bootstrapDns = selector(state, 'bootstrap_dns');
    const allServers = selector(state, 'all_servers');
    return {
        upstreamDns,
        bootstrapDns,
        allServers,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({ form: 'upstreamForm' }),
])(Form);
