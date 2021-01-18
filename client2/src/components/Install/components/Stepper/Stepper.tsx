import React, { FC } from 'react';
import cn from 'classnames';

import s from './Stepper.module.pcss';

interface StepProps {
    active: boolean;
    current: boolean;
}

const Step: FC<StepProps> = ({ active, current }) => {
    return (
        <div className={cn(s.wrap, { [s.active]: active, [s.current]: current })}>
            <div className={s.circle} />
        </div>
    );
};

interface StepperProps {
    currentStep: number;
}

const Stepper: FC<StepperProps> = ({ currentStep }) => {
    return (
        <div className={s.stepper}>
            <Step current={currentStep === 0} active={currentStep >= 0} />
            <Step current={currentStep === 1} active={currentStep >= 1} />
            <Step current={currentStep === 2} active={currentStep >= 2} />
            <Step current={currentStep === 3} active={currentStep >= 3} />
            <Step current={currentStep === 4} active={currentStep >= 4} />
        </div>
    );
};

export default Stepper;
