import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

function Version(props) {
    const { dnsVersion, dnsAddress, dnsPort } = props;
    return (
        <div className="nav-version">
            <div className="nav-version__text">
                <Trans>version</Trans>: <span className="nav-version__value">{dnsVersion}</span>
            </div>
            <div className="nav-version__text">
                <Trans>address</Trans>: <span className="nav-version__value">{dnsAddress}:{dnsPort}</span>
            </div>
        </div>
    );
}

Version.propTypes = {
    dnsVersion: PropTypes.string,
    dnsAddress: PropTypes.string,
    dnsPort: PropTypes.number,
};

export default withNamespaces()(Version);
