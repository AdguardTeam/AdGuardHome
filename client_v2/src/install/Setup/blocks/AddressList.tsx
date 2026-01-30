import React from 'react';

import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';
import { getInterfaceIp } from 'panel/helpers/helpers';
import { ALL_INTERFACES_IP } from 'panel/helpers/constants';
import { InstallInterface } from 'panel/initialState';
import styles from '../styles.module.pcss';
import { stripZoneId, getDnsAddressWithPort } from '../helpers/helpers';

interface renderItemProps {
    ip: string;
    port: number;
    isDns: boolean;
    interfaceName?: string;
}

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
        <li className={styles.addressListItem}
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

export const AddressList = ({ address, interfaces, port, isDns }: AddressListProps) => (
    <ul className={styles.addressList}>
        {address === ALL_INTERFACES_IP
            ? Object.values(interfaces)
                  .filter((iface: InstallInterface) => iface?.ip_addresses?.length > 0)
                  .sort((a: InstallInterface, b: InstallInterface) => {
                      const aIp = stripZoneId(getInterfaceIp(a));
                      const bIp = stripZoneId(getInterfaceIp(b));

                      const ipCompare = aIp.localeCompare(bIp);
                      if (ipCompare !== 0) {
                          return ipCompare;
                      }

                      return (a.name ?? '').localeCompare(b.name ?? '');
                  })
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
