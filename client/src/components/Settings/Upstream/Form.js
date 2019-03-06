import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Field, reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';
import classnames from 'classnames';

import { renderSelectField } from '../../../helpers/form';

let Form = (props) => {
    const {
        t,
        handleSubmit,
        testUpstream,
        upstreamDns,
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
                        <label>{t('upstream_dns')}</label>
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
                    <div className="form__group">
                        <label>{t('bootstrap_dns')}</label>
                        <Field
                            id="bootstrap_dns"
                            name="bootstrap_dns"
                            component="textarea"
                            type="text"
                            className="form-control"
                            placeholder={t('bootstrap_dns_desc')}
                        />
                    </div>
                </div>
            </div>
            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="button"
                        className={testButtonClass}
                        onClick={() => testUpstream(upstreamDns)}
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
    processingTestUpstream: PropTypes.bool,
    processingSetUpstream: PropTypes.bool,
    t: PropTypes.func,
};

const selector = formValueSelector('upstreamForm');

Form = connect((state) => {
    const upstreamDns = selector(state, 'upstream_dns');
    return {
        upstreamDns,
    };
})(Form);

export default flow([
    withNamespaces(),
    reduxForm({ form: 'upstreamForm' }),
])(Form);
