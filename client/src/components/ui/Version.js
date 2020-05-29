import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import './Version.css';

const Version = (props) => {
    const {
        dnsVersion = 'undefined', processingVersion, t, checkUpdateFlag,
    } = props;

    return (
        <div className="version">
            <div className="version__text">
                <Trans>version</Trans>:&nbsp;
                <span className="version__value" title={dnsVersion}>{dnsVersion}</span>
                {checkUpdateFlag && <button
                    type="button"
                    className="btn btn-icon btn-icon-sm btn-outline-primary btn-sm ml-2"
                    onClick={() => props.getVersion(true)}
                    disabled={processingVersion}
                    title={t('check_updates_now')}
                >
                    <svg className="icons">
                        <use xlinkHref="#refresh" />
                    </svg>
                </button>}
            </div>
        </div>
    );
};

Version.propTypes = {
    dnsVersion: PropTypes.string.isRequired,
    getVersion: PropTypes.func.isRequired,
    processingVersion: PropTypes.bool.isRequired,
    checkUpdateFlag: PropTypes.bool.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(Version);
