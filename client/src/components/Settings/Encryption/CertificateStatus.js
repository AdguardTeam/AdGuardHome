import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { withTranslation, Trans } from 'react-i18next';
import format from 'date-fns/format';

import { EMPTY_DATE } from '../../../helpers/constants';

const CertificateStatus = ({
    validChain,
    validCert,
    subject,
    issuer,
    notAfter,
    dnsNames,
}) => (
    <Fragment>
        <div className="form__label form__label--bold">
            <Trans>encryption_status</Trans>:
        </div>
        <ul className="encryption__list">
            <li
                className={validChain ? 'text-success' : 'text-danger'}
            >
                {validChain ? (
                    <Trans>encryption_chain_valid</Trans>
                ) : (
                    <Trans>encryption_chain_invalid</Trans>
                )}
            </li>
            {validCert && (
                <Fragment>
                    {subject && (
                        <li>
                            <Trans>encryption_subject</Trans>:&nbsp;
                            {subject}
                        </li>
                    )}
                    {issuer && (
                        <li>
                            <Trans>encryption_issuer</Trans>:&nbsp;
                            {issuer}
                        </li>
                    )}
                    {notAfter && notAfter !== EMPTY_DATE && (
                        <li>
                            <Trans>encryption_expire</Trans>:&nbsp;
                            {format(notAfter, 'YYYY-MM-DD HH:mm:ss')}
                        </li>
                    )}
                    {dnsNames && (
                        <li>
                            <Trans>encryption_hostnames</Trans>:&nbsp;
                            {dnsNames.join(', ')}
                        </li>
                    )}
                </Fragment>
            )}
        </ul>
    </Fragment>
);

CertificateStatus.propTypes = {
    validChain: PropTypes.bool.isRequired,
    validCert: PropTypes.bool.isRequired,
    subject: PropTypes.string,
    issuer: PropTypes.string,
    notAfter: PropTypes.string,
    dnsNames: PropTypes.arrayOf(PropTypes.string),
};

export default withTranslation()(CertificateStatus);
