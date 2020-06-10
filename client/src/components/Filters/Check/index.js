import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import { Field, reduxForm } from 'redux-form';
import flow from 'lodash/flow';
import Card from '../../ui/Card';

import { renderInputField } from '../../../helpers/form';
import Info from './Info';
import { FORM_NAME } from '../../../helpers/constants';

const Check = (props) => {
    const {
        t,
        handleSubmit,
        pristine,
        invalid,
        processing,
        check,
        filters,
        whitelistFilters,
    } = props;

    const {
        hostname,
        reason,
        filter_id,
        rule,
        service_name,
        cname,
        ip_addrs,
    } = check;

    return (
        <Card
            title={t('check_title')}
            subtitle={t('check_desc')}
        >
            <form onSubmit={handleSubmit}>
                <div className="row">
                    <div className="col-12 col-md-6">
                        <div className="input-group">
                            <Field
                                id="name"
                                name="name"
                                component={renderInputField}
                                type="text"
                                className="form-control"
                                placeholder={t('form_enter_host')}
                            />
                            <span className="input-group-append">
                                <button
                                    className="btn btn-success btn-standard btn-large"
                                    type="submit"
                                    onClick={handleSubmit}
                                    disabled={pristine || invalid || processing}
                                >
                                    <Trans>check</Trans>
                                </button>
                            </span>
                        </div>
                        {check.hostname && (
                            <Fragment>
                                <hr />
                                <Info
                                    filters={filters}
                                    whitelistFilters={whitelistFilters}
                                    hostname={hostname}
                                    reason={reason}
                                    filter_id={filter_id}
                                    rule={rule}
                                    service_name={service_name}
                                    cname={cname}
                                    ip_addrs={ip_addrs}
                                />
                            </Fragment>
                        )}
                    </div>
                </div>
            </form>
        </Card>
    );
};

Check.propTypes = {
    t: PropTypes.func.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    pristine: PropTypes.bool.isRequired,
    invalid: PropTypes.bool.isRequired,
    processing: PropTypes.bool.isRequired,
    check: PropTypes.object.isRequired,
    filters: PropTypes.array.isRequired,
    whitelistFilters: PropTypes.array.isRequired,
};

export default flow([
    withTranslation(),
    reduxForm({ form: FORM_NAME.DOMAIN_CHECK }),
])(Check);
