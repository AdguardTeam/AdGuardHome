import React from 'react';

import { getIpList, getDnsAddress, getWebAddress } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';
import { DhcpInterface } from '../../initialState';

interface renderItemProps {
    ip: string;
    port: number;
    isDns: boolean;
}

const renderItem = ({ ip, port, isDns }: renderItemProps) => {
    const webAddress = getWebAddress(ip, port);
    const dnsAddress = getDnsAddress(ip, port);

    return (
        <li key={ip}>
            {isDns ? (
                <strong>{dnsAddress}</strong>
            ) : (
                <a href={webAddress} target="_blank" rel="noopener noreferrer">
                    {webAddress}
                </a>
            )}
        </li>
    );
};

interface AddressListProps {
    interfaces: DhcpInterface[];
    address: string;
    port: number;
    isDns?: boolean;
}

const AddressList = ({ address, interfaces, port, isDns }: AddressListProps) => (
    <ul className="list-group pl-4">
        {address === ALL_INTERFACES_IP
            ? getIpList(interfaces).map((ip: any) =>
                  renderItem({
                      ip,
                      port,
                      isDns,
                  }),
              )
            : renderItem({
                  ip: address,
                  port,
                  isDns,
              })}
    </ul>
);

export default AddressList;
