import React from 'react';

import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import Guide from '../../components/ui/Guide';

import Controls from './Controls';

import AddressList from './AddressList';
import { DhcpInterface } from '../../initialState';

interface DevicesProps {
    interfaces: DhcpInterface[];
    dnsIp: string;
    dnsPort: number;
}

const Devices = ({ interfaces, dnsIp, dnsPort }: DevicesProps) => (
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
                    <AddressList interfaces={interfaces} address={dnsIp} port={dnsPort} isDns />
                </div>
            </div>

            <Guide />
        </div>

        <Controls />
    </div>
);

export default flow([
    withTranslation(),
])(Devices);
