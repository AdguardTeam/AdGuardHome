import React from 'react';
import { Trans, withTranslation } from 'react-i18next';

import { Guide } from '@/components/ui/Guide';

import Card from '@/components/ui/Card';

import PageTitle from '@/components/ui/PageTitle';
import './Guide.css';
import { DashboardData } from '@/initialState';

interface SetupGuideProps {
    dashboard: DashboardData;
    t: (id: string) => string;
}

const SetupGuide = ({ t, dashboard: { dnsAddresses } }: SetupGuideProps) => (
    <div className="guide">
        <PageTitle title={t('setup_guide')} />

        <Card>
            <div className="guide__title">
                <Trans>install_devices_title</Trans>
            </div>

            <div className="guide__desc">
                <Trans>install_devices_desc</Trans>

                <div className="mt-1">
                    <Trans>install_devices_address</Trans>:
                </div>

                <ul className="guide__list">
                    {dnsAddresses.map((ip: any) => (
                        <li key={ip} className="guide__address">
                            {ip}
                        </li>
                    ))}
                </ul>
            </div>

            <Guide dnsAddresses={dnsAddresses} />
        </Card>
    </div>
);

export default withTranslation()(SetupGuide);
