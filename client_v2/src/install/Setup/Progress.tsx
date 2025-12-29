import React from 'react';
import intl from "panel/common/intl"
import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';

type Props = { step: number };

export const Progress = ({ step }: Props) => (
    <div className="setup__progress">
        <div className="setup__progress-text">
            <div className="setup__step-message">
                {intl.getMessage("install_step")}
            </div>
            {step}/{INSTALL_TOTAL_STEPS}
        </div>

        <div className="setup__progress-wrap" role="progressbar" aria-valuenow={step} aria-valuemin={1} aria-valuemax={INSTALL_TOTAL_STEPS}>
            {Array.from({ length: INSTALL_TOTAL_STEPS }, (_, i) => {
                const installStep = i + 1;
                const isDoneOrCurrent = installStep <= step;

                return (
                    <div
                        key={installStep}
                        className={`setup__progress-step ${isDoneOrCurrent ? 'setup__progress-step--green' : 'setup__progress-step--grey'}`}
                    />
                );
            })}
        </div>
    </div>
);
