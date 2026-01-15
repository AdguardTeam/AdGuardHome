import React from 'react';
import intl from "panel/common/intl"
import { INSTALL_TOTAL_STEPS } from '../../helpers/constants';
import setup from './styles.module.pcss'

type Props = { step: number };

export const Progress = ({ step }: Props) => {
    const totalProgressSteps = INSTALL_TOTAL_STEPS - 1;
    const progressStep = Math.min(step, totalProgressSteps);

    if (step >= INSTALL_TOTAL_STEPS) {
        return null;
    }

    return (
        <div className={setup.progress}>
            <div>
                <div className={setup.message}>
                    {intl.getMessage("install_step")}
                </div>
                {progressStep}/{totalProgressSteps}
            </div>

            <div
                className={setup.progressWrap}
                role="progressbar"
                aria-valuenow={progressStep}
                aria-valuemin={1}
                aria-valuemax={totalProgressSteps}
            >
                {Array.from({ length: totalProgressSteps }, (_, i) => {
                    const installStep = i + 1;
                    const isDoneOrCurrent = installStep <= progressStep;

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
