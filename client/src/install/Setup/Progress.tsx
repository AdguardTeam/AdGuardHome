import React from 'react';
import { Trans, withTranslation } from 'react-i18next';

import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';

const getProgressPercent = (step: any) => (step / INSTALL_TOTAL_STEPS) * 100;

type Props = {
    step: number;
};

const Progress = (props: Props) => (
    <div className="setup__progress">
        <Trans>install_step</Trans> {props.step}/{INSTALL_TOTAL_STEPS}
        <div className="setup__progress-wrap">
            <div className="setup__progress-inner" style={{ width: `${getProgressPercent(props.step)}%` }} />
        </div>
    </div>
);

export default withTranslation()(Progress);
