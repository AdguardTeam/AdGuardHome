import React from 'react';
import PropTypes from 'prop-types';

import { getIpList, getAddress } from '../../helpers/helpers';

const AddressList = (props) => {
    let webAddress = getAddress(props.address, props.port);
    let dnsAddress = getAddress(props.address, props.port, true);

    if (props.address === '0.0.0.0') {
        return getIpList(props.interfaces).map((ip) => {
            webAddress = getAddress(ip, props.port);
            dnsAddress = getAddress(ip, props.port, true);

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
