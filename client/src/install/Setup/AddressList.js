import React from 'react';
import PropTypes from 'prop-types';

import { getIpList, getDnsAddress, getWebAddress } from '../../helpers/helpers';
import { ALL_INTERFACES_IP } from '../../helpers/constants';

const AddressList = (props) => {
    let webAddress = getWebAddress(props.address, props.port);
    let dnsAddress = getDnsAddress(props.address, props.port);

    if (props.address === ALL_INTERFACES_IP) {
        return getIpList(props.interfaces).map((ip) => {
            webAddress = getWebAddress(ip, props.port);
            dnsAddress = getDnsAddress(ip, props.port);

            if (props.isDns) {
                return (
                    <li key={ip}>
                        <strong>
                            {dnsAddress}
                        </strong>
                    </li>
                );
            }

            return (
                <li key={ip}>
                    <a href={webAddress}>
                        {webAddress}
                    </a>
                </li>
            );
        });
    }

    if (props.isDns) {
        return (
            <strong>
                {dnsAddress}
            </strong>
        );
    }

    return (
        <a href={webAddress}>
            {webAddress}
        </a>
    );
};

AddressList.propTypes = {
    interfaces: PropTypes.object.isRequired,
    address: PropTypes.string.isRequired,
    port: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    isDns: PropTypes.bool,
};

export default AddressList;
