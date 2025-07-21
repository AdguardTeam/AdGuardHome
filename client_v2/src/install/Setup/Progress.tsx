import React from 'react';
import { Trans } from 'react-i18next';

import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';

const getProgressPercent = (step: number) => (step / INSTALL_TOTAL_STEPS) * 100;

type Props = {
    step: number;
};

export const Progress = ({ step }: Props) => (
    <div className="setup__progress">
        <Trans>install_step</Trans> {step}/{INSTALL_TOTAL_STEPS}
        <div className="setup__progress-wrap">
            <div className="setup__progress-inner" style={{ width: `${getProgressPercent(step)}%` }} />
        </div>
    </div>
);
