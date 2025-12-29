import React from 'react';

import { Guide } from 'panel/common/ui/Guide/Guide';
import intl from 'panel/common/intl';
import Controls from './Controls';

type Props = {
    dnsAddresses?: string[];
};

export const SetupGuideStep = ({ dnsAddresses = [] }: Props) => {
    return (
        <div className="setup__group--center">
            <div className="setup__subtitle">
                <div className="setup__title">{intl.getMessage('setup_guide_title')}</div>
            </div>

            <Guide dnsAddresses={dnsAddresses} />

            <Controls />
        </div>
    );
};
