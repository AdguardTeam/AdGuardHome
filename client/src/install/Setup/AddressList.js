import React from 'react';
import PropTypes from 'prop-types';

import { getIpList, getDnsAddress, getWebAddress } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';

const renderItem = ({
    ip, port, isDns,
}) => {
    const webAddress = getWebAddress(ip, port);
    const dnsAddress = getDnsAddress(ip, port);

    return <li key={ip}>{isDns
        ? <strong>{dnsAddress}</strong>
        : <a href={webAddress}>{webAddress}</a>
    }
    </li>;
};

const AddressList = ({
    address,
    interfaces,
    port,
    isDns,
}) => <ul className="list-group pl-4">{
    address === ALL_INTERFACES_IP
        ? getIpList(interfaces)
            .map((ip) => renderItem({
                ip,
                port,
                isDns,
            }))
        : renderItem({
            ip: address,
            port,
            isDns,
        })
}
</ul>;

AddressList.propTypes = {
    interfaces: PropTypes.object.isRequired,
    address: PropTypes.string.isRequired,
    port: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    isDns: PropTypes.bool,
};

renderItem.propTypes = {
    ip: PropTypes.string.isRequired,
    port: PropTypes.string.isRequired,
    isDns: PropTypes.bool.isRequired,
};

export default AddressList;
