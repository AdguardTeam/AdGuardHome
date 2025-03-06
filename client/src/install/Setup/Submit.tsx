import React from 'react';

import { Trans } from 'react-i18next';

import Controls from './Controls';
import { WebConfig } from './Settings';

type Props = {
    webConfig: WebConfig;
    openDashboard: (ip: string, port: number) => void;
};

export const Submit = ({ openDashboard, webConfig }: Props) => (
    <div className="setup__step">
        <div className="setup__group">
            <h1 className="setup__title">
                <Trans>install_submit_title</Trans>
            </h1>

            <p className="setup__desc">
                <Trans>install_submit_desc</Trans>
            </p>
        </div>

        <Controls openDashboard={openDashboard} ip={webConfig.ip} port={webConfig.port} />
    </div>
);
