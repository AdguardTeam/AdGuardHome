import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm, formValueSelector } from 'redux-form';
import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import Guide from '../../components/ui/Guide';
import Controls from './Controls';
import AddressList from './AddressList';
import { FORM_NAME } from '../../helpers/constants';

let Devices = (props) => (
    <div className="setup__step">
        <div className="setup__group">
            <div className="setup__subtitle">
                <Trans>install_devices_title</Trans>
            </div>
            <div className="setup__desc">
                <Trans>install_devices_desc</Trans>
                <div className="mt-1">
                    <Trans>install_devices_address</Trans>:
                </div>
                <div className="mt-1">
                    <AddressList
                        interfaces={props.interfaces}
                        address={props.dnsIp}
                        port={props.dnsPort}
                        isDns={true}
                    />
                </div>
            </div>
            <Guide />
        </div>
        <Controls />
    </div>
);

Devices.propTypes = {
    interfaces: PropTypes.object.isRequired,
    dnsIp: PropTypes.string.isRequired,
    dnsPort: PropTypes.number.isRequired,
};

const selector = formValueSelector('install');

Devices = connect((state) => {
    const dnsIp = selector(state, 'dns.ip');
    const dnsPort = selector(state, 'dns.port');

    return {
        dnsIp,
        dnsPort,
    };
})(Devices);

export default flow([
    withTranslation(),
    reduxForm({
        form: FORM_NAME.INSTALL,
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(Devices);
