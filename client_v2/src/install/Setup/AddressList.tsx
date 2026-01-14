import React from 'react';

import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';
import { getInterfaceIp } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';
import { InstallInterface } from '../../initialState';
import setup from './styles.module.pcss'
import { stripZoneId } from './helpers'

interface renderItemProps {
    ip: string;
    port: number;
    isDns: boolean;
    interfaceName?: string;
}

const getDnsAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `[${normalizedIp}]:${port}`;
    }

    return `${normalizedIp}:${port}`;
};

const getWebAddressWithPort = (ip: string, port: number) => {
    const normalizedIp = stripZoneId(ip);

    if (normalizedIp.includes(':') && !normalizedIp.includes('[')) {
        return `http://[${normalizedIp}]:${port}`;
    }

    return `http://${normalizedIp}:${port}`;
};

const renderItem = ({ ip, port, isDns }: renderItemProps) => {
    const webAddress = getWebAddressWithPort(ip, port);
    const dnsAddress = getDnsAddressWithPort(ip, port);

    return (
        <li className={setup.addressListItem}
            key={ip}>
            {isDns ? <CopiedText text={dnsAddress} /> : <CopiedText text={webAddress} />}
        </li>
    );
};

interface AddressListProps {
    interfaces: InstallInterface[];
    address: string;
    port: number;
    isDns?: boolean;
}

const AddressList = ({ address, interfaces, port, isDns }: AddressListProps) => (
    <ul className={setup.addressList}>
        {address === ALL_INTERFACES_IP
            ? Object.values(interfaces)
                  .filter((iface: InstallInterface) => iface?.ip_addresses?.length > 0)
                  .map((iface: InstallInterface) => {
                      const ip = getInterfaceIp(iface);

                      return renderItem({
                          ip,
                          port,
                          isDns,
                          interfaceName: iface.name,
                      });
                  })
            : renderItem({
                  ip: address,
                  port,
                  isDns,
              })}
    </ul>
);

export default AddressList;
