import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import { getDnsAddress } from '../../helpers/helpers';

const Version = (props) => {
    const {
        dnsVersion, dnsAddresses, dnsPort, processingVersion, t,
    } = props;

    return (
        <div className="nav-version">
            <div className="nav-version__text">
                <Trans>version</Trans>:&nbsp;<span className="nav-version__value" title={dnsVersion}>{dnsVersion}</span>
                <button
                    type="button"
                    className="btn btn-icon btn-icon-sm btn-outline-primary btn-sm ml-2"
                    onClick={() => props.getVersion(true)}
                    disabled={processingVersion}
                    title={t('check_updates_now')}
                >
                    <svg className="icons">
                        <use xlinkHref="#refresh" />
                    </svg>
                </button>
            </div>
            <div className="nav-version__link">
                <div className="popover__trigger popover__trigger--address">
                    <Trans>dns_addresses</Trans>
                </div>
                <div className="popover__body popover__body--address">
                    <div className="popover__list">
                        {dnsAddresses.map(ip => (
                            <li key={ip}>{getDnsAddress(ip, dnsPort)}</li>
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
};

Version.propTypes = {
    dnsVersion: PropTypes.string.isRequired,
    dnsAddresses: PropTypes.array.isRequired,
    dnsPort: PropTypes.number.isRequired,
    getVersion: PropTypes.func.isRequired,
    processingVersion: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Version);
