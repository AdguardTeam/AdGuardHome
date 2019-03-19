import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import { getDnsAddress } from '../../helpers/helpers';

import Guide from '../ui/Guide';
import Card from '../ui/Card';
import PageTitle from '../ui/PageTitle';
import './Guide.css';

const SetupGuide = ({
    t,
    dashboard: {
        dnsAddresses,
        dnsPort,
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
                <div className="mt-2 font-weight-bold">
                    {dnsAddresses
                        .map(ip => <li key={ip}>{getDnsAddress(ip, dnsPort)}</li>)
                    }
                </div>
            </div>
            <Guide />
        </Card>
    </div>
);

SetupGuide.propTypes = {
    dashboard: PropTypes.object.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(SetupGuide);
