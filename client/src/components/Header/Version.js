import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import { getDnsAddress } from '../../helpers/helpers';

function Version(props) {
    const { dnsVersion, dnsAddresses, dnsPort } = props;
    return (
        <div className="nav-version">
            <div className="nav-version__text">
                <Trans>version</Trans>: <span className="nav-version__value">{dnsVersion}</span>
            </div>
            <div className="nav-version__link">
                <div className="popover__trigger popover__trigger--address">
                    <Trans>dns_addresses</Trans>
                </div>
                <div className="popover__body popover__body--address">
                    <div className="popover__list">
                        {dnsAddresses
                            .map(ip => <li key={ip}>{getDnsAddress(ip, dnsPort)}</li>)
                        }
                    </div>
                </div>
            </div>
        </div>
    );
}

Version.propTypes = {
    dnsVersion: PropTypes.string.isRequired,
    dnsAddresses: PropTypes.array.isRequired,
    dnsPort: PropTypes.number.isRequired,
};

export default withNamespaces()(Version);
