import React from 'react';

import intl from 'panel/common/intl';
import Controls from './Controls';
import { WebConfig } from './Settings';

type Props = {
    webConfig: WebConfig;
    openDashboard: (ip: string, port: number) => void;
};

export const Submit = ({ openDashboard, webConfig }: Props) => (
    <div className="setup__end">
        <div className="setup__group">
            <h1 className="setup__title">{intl.getMessage('install_submit_title')}</h1>

            <p className="setup__desc">
                {intl.getMessage('setup_complete')}
            </p>
        </div>

        <Controls openDashboard={openDashboard} ip={webConfig.ip} port={webConfig.port} />
    </div>
);
