import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';
import { FORM_NAME } from '../../helpers/constants';

let Submit = (props) => (
    <div className="setup__step">
        <div className="setup__group">
            <h1 className="setup__title">
                <Trans>install_submit_title</Trans>
            </h1>
            <p className="setup__desc">
                <Trans>install_submit_desc</Trans>
            </p>
        </div>
        <Controls
            openDashboard={props.openDashboard}
            ip={props.webIp}
            port={props.webPort}
        />
    </div>
);

Submit.propTypes = {
    webIp: PropTypes.string.isRequired,
    webPort: PropTypes.number.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    pristine: PropTypes.bool.isRequired,
    submitting: PropTypes.bool.isRequired,
    openDashboard: PropTypes.func.isRequired,
};

const selector = formValueSelector('install');

Submit = connect((state) => {
    const webIp = selector(state, 'web.ip');
    const webPort = selector(state, 'web.port');

    return {
        webIp,
        webPort,
    };
})(Submit);


export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.INSTALL,
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(Submit);
