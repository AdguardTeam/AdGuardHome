import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import Guide from '../ui/Guide';
import Card from '../ui/Card';
import PageTitle from '../ui/PageTitle';
import './Guide.css';

const SetupGuide = ({
    t,
    dashboard: {
        dnsAddresses,
    },
}) => (
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
                <div className="mt-3">
                    {dnsAddresses.map((ip) => <li key={ip} className="guide__address">{ip}</li>)}
                </div>
            </div>
            <Guide dnsAddresses={dnsAddresses} />
        </Card>
    </div>
);

SetupGuide.propTypes = {
    dashboard: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(SetupGuide);
