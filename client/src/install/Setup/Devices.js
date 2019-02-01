import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm, formValueSelector } from 'redux-form';
import { Trans, withNamespaces } from 'react-i18next';
import flow from 'lodash/flow';

import Tabs from '../../components/ui/Tabs';
import Icons from '../../components/ui/Icons';
import Controls from './Controls';

let Devices = props => (
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
                <div>
                    <strong>{`${props.dnsIp}:${props.dnsPort}`}</strong>
                </div>
            </div>
            <Icons />
            <Tabs>
                <div label="Router">
                    <div className="tab__title">
                        <Trans>install_devices_router</Trans>
                    </div>
                    <div className="tab__text">
                        <p><Trans>install_devices_router_desc</Trans></p>
                        <ol>
                            <li><Trans>install_devices_router_list_1</Trans></li>
                            <li><Trans>install_devices_router_list_2</Trans></li>
                            <li><Trans>install_devices_router_list_3</Trans></li>
                        </ol>
                    </div>
                </div>
                <div label="Windows">
                    <div className="tab__title">
                        Windows
                    </div>
                    <div className="tab__text">
                        <ol>
                            <li><Trans>install_devices_windows_list_1</Trans></li>
                            <li><Trans>install_devices_windows_list_2</Trans></li>
                            <li><Trans>install_devices_windows_list_3</Trans></li>
                            <li><Trans>install_devices_windows_list_4</Trans></li>
                            <li><Trans>install_devices_windows_list_5</Trans></li>
                            <li><Trans>install_devices_windows_list_6</Trans></li>
                        </ol>
                    </div>
                </div>
                <div label="macOS">
                    <div className="tab__title">
                        macOS
                    </div>
                    <div className="tab__text">
                        <ol>
                            <li><Trans>install_devices_macos_list_1</Trans></li>
                            <li><Trans>install_devices_macos_list_2</Trans></li>
                            <li><Trans>install_devices_macos_list_3</Trans></li>
                            <li><Trans>install_devices_macos_list_4</Trans></li>
                        </ol>
                    </div>
                </div>
                <div label="Android">
                    <div className="tab__title">
                        Android
                    </div>
                    <div className="tab__text">
                        <ol>
                            <li><Trans>install_devices_android_list_1</Trans></li>
                            <li><Trans>install_devices_android_list_2</Trans></li>
                            <li><Trans>install_devices_android_list_3</Trans></li>
                            <li><Trans>install_devices_android_list_4</Trans></li>
                            <li><Trans>install_devices_android_list_5</Trans></li>
                        </ol>
                    </div>
                </div>
                <div label="iOS">
                    <div className="tab__title">
                        iOS
                    </div>
                    <div className="tab__text">
                        <ol>
                            <li><Trans>install_devices_ios_list_1</Trans></li>
                            <li><Trans>install_devices_ios_list_2</Trans></li>
                            <li><Trans>install_devices_ios_list_3</Trans></li>
                            <li><Trans>install_devices_ios_list_4</Trans></li>
                        </ol>
                    </div>
                </div>
            </Tabs>
        </div>
        <Controls />
    </div>
);

Devices.propTypes = {
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
    withNamespaces(),
    reduxForm({
        form: 'install',
        destroyOnUnmount: false,
        forceUnregisterOnUnmount: true,
    }),
])(Devices);
