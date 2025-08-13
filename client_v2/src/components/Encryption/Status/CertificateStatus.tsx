import React from 'react';
import format from 'date-fns/format';

import { EMPTY_DATE } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';
import s from './styles.module.pcss';

type Props = {
    validChain: boolean;
    validCert: boolean;
    subject?: string;
    issuer?: string;
    notAfter?: string;
    dnsNames?: string[];
};

export const CertificateStatus = ({ validChain, validCert, subject, issuer, notAfter, dnsNames }: Props) => (
    <StatusBlock
        variant={validChain ? 'success' : 'error'}
        title={validChain ? intl.getMessage('encryption_chain_valid') : intl.getMessage('encryption_chain_invalid')}
    >
        {validCert && (subject || issuer || notAfter || dnsNames) && (
            <ul className={s.statusList}>
                {subject && <li>{intl.getMessage('encryption_subject', { value: subject })}</li>}
                {issuer && <li>{intl.getMessage('encryption_issuer', { value: issuer })}</li>}
                {notAfter && notAfter !== EMPTY_DATE && (
                    <li>{intl.getMessage('encryption_expire', { value: format(notAfter, 'YYYY-MM-DD HH:mm:ss') })}</li>
                )}
                {dnsNames && <li>{intl.getMessage('encryption_hostnames', { value: dnsNames.join(', ') })}</li>}
            </ul>
        )}
    </StatusBlock>
);
