import React from 'react';
import intl from 'panel/common/intl';
import cn from 'clsx';
import { INSTALL_TOTAL_STEPS } from 'panel/helpers/constants';
import styles from './styles.module.pcss';

type Props = { step: number };

export const Progress = ({ step }: Props) => {
    const totalProgressSteps = INSTALL_TOTAL_STEPS - 1;
    const progressStep = Math.min(step, totalProgressSteps);

    if (step >= INSTALL_TOTAL_STEPS) {
        return null;
    }

    return (
        <div className={styles.progress}>
            <div>
                <div className={styles.message}>
                    {intl.getMessage('install_step')}
                </div>
                {progressStep}/{totalProgressSteps}
            </div>

            <div
                className={styles.progressWrap}
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
                            className={cn(styles.progressStep, {
                                [styles.progressStepGreen]: isDoneOrCurrent,
                                [styles.progressStepGrey]: !isDoneOrCurrent,
                            })}
                        />
                    );
                })}
            </div>
        </div>
    );
};
