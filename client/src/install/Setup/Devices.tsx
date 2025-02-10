import React from 'react';

import { Trans } from 'react-i18next';

import { Guide } from '../../components/ui/Guide';

import Controls from './Controls';

import AddressList from './AddressList';
import { InstallInterface } from '../../initialState';
import { DnsConfig } from './Settings';

type Props = {
    interfaces: InstallInterface[];
    dnsConfig: DnsConfig;
};

export const Devices = ({ interfaces, dnsConfig }: Props) => (
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
                    <AddressList interfaces={interfaces} address={dnsConfig.ip} port={dnsConfig.port} isDns />
                </div>
            </div>

            <Guide />
        </div>

        <Controls />
    </div>
);
