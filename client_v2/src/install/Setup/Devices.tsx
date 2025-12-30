import React from 'react';

import intl from 'panel/common/intl';

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
                {intl.getMessage('install_devices_title')}
            </div>

            <div className="setup__desc">
                {intl.getMessage('install_devices_desc')}

                <div className="mt-1">
                    {intl.getMessage('install_devices_address')}:
                </div>

                <div className="mt-1">
                    <AddressList interfaces={interfaces} address={dnsConfig.ip} port={dnsConfig.port} isDns />
                </div>
            </div>
        </div>

        <Controls />
    </div>
);
