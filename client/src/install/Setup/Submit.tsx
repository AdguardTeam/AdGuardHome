import React from 'react';

import { Trans, withTranslation } from 'react-i18next';
import flow from 'lodash/flow';

import Controls from './Controls';

interface SubmitProps {
    webIp: string;
    webPort: number;
    handleSubmit: (...args: unknown[]) => string;
    pristine: boolean;
    submitting: boolean;
    openDashboard: (...args: unknown[]) => unknown;
}

const Submit = (props: SubmitProps) => (
    <div className="setup__step">
        <div className="setup__group">
            <h1 className="setup__title">
                <Trans>install_submit_title</Trans>
            </h1>

            <p className="setup__desc">
                <Trans>install_submit_desc</Trans>
            </p>
        </div>

        <Controls openDashboard={props.openDashboard} ip={props.webIp} port={props.webPort} />
    </div>
);

export default flow([withTranslation()])(Submit);
