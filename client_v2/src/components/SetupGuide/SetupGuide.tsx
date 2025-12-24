import React from 'react';
import { shallowEqual, useSelector } from 'react-redux';

import { RootState } from 'panel/initialState';
import { Guide } from 'panel/common/ui/Guide/Guide';

import theme from 'panel/lib/theme';

import intl from 'panel/common/intl';
import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';
import s from './SetupGuide.module.pcss';

export const SetupGuide = () => {
    const { dnsAddresses } = useSelector((state: RootState) => state.dashboard, shallowEqual);

    const encryptedAddresses = dnsAddresses.filter((address: string) =>
        address.includes('https://') || address.includes('tls://') || address.includes('quic://')
    );
    const plainAddresses = dnsAddresses.filter((address: string) =>
        !address.includes('https://') && !address.includes('tls://') && !address.includes('quic://')
    );

    return (
        <div className={theme.layout.container}>
            <div className={s.header}>
                <h1 className={s.pageTitle}>{intl.getMessage('setup_guide')}</h1>
                <div className={s.pageDesc}>{intl.getMessage('setup_guide_desc')}</div>
            </div>

            <div className={s.guidePage}>
                <h1 className={s.guideTitle}>{intl.getMessage('setup_guide_device_type')}</h1>
                <Guide dnsAddresses={dnsAddresses} />

                <div className={s.guideDesc}>
                    <h1 className={s.dnsTitle}>{intl.getMessage('home_dns_addresses')}</h1>

                    <p>{intl.getMessage('home_dns_addresses_desc')}</p>

                    {encryptedAddresses.length > 0 && (
                        <>
                            <div className={s.dnsSubtitle}>
                                {intl.getMessage('encrypted_dns_addresses')}
                            </div>

                            <ul className={s.addressList}>
                                {encryptedAddresses.map((ip: string) => (
                                    <li key={ip} className={s.address}>
                                        <span className={s.bulletIcon}></span>
                                        <CopiedText text={ip} />
                                    </li>
                                ))}
                            </ul>
                        </>
                    )}

                    {plainAddresses.length > 0 && (
                        <>
                            <div className={s.dnsSubtitle}>
                                {intl.getMessage('plain_dns_addresses')}
                            </div>

                            <ul className={s.addressList}>
                                {plainAddresses.map((ip: string) => (
                                    <li key={ip} className={s.address}>
                                        <span className={s.bulletIcon}></span>
                                        <CopiedText text={ip} />
                                    </li>
                                ))}
                            </ul>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};
