import React from 'react';
import intl from "panel/common/intl"
import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';
import setup from './styles.module.pcss'

type Props = { step: number };

export const Progress = ({ step }: Props) => {
    if (step > INSTALL_TOTAL_STEPS) {
        return null;
    }

    return (
        <div className={setup.progress}>
            <div>
                <div className={setup.message}>
                    {intl.getMessage("install_step")}
                </div>
                {step}/{INSTALL_TOTAL_STEPS}
            </div>

            <div className={setup.progressWrap} role="progressbar" aria-valuenow={step} aria-valuemin={1} aria-valuemax={INSTALL_TOTAL_STEPS}>
                {Array.from({ length: INSTALL_TOTAL_STEPS }, (_, i) => {
                    const installStep = i + 1;
                    const isDoneOrCurrent = installStep <= step;

                    return (
                        <div
                            key={installStep}
                            className={`${setup.progressStep} ${
                                isDoneOrCurrent ? setup.progressStepGreen : setup.progressStepGrey
                            }`}
                        />
                    );
                })}
            </div>
        </div>
    );
};
