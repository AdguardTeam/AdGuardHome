import React from 'react';
import PropTypes from 'prop-types';

export default function Version(props) {
    const { dnsVersion, dnsAddress, dnsPort } = props;
    return (
        <div className="nav-version">
            version {dnsVersion} / address: {dnsAddress}:{dnsPort}
        </div>
    );
}

Version.propTypes = {
    dnsVersion: PropTypes.string,
    dnsAddress: PropTypes.string,
    dnsPort: PropTypes.number,
};
