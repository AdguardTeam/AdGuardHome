import React, { FC } from 'react';
import { Steps } from 'antd';

import s from './Stepper.module.pcss';

interface StepperProps {
    currentStep: number;
}

const { Step } = Steps;

const Stepper: FC<StepperProps> = ({ currentStep }) => {
    return (
        <Steps progressDot current={currentStep} className={s.stepper}>
            <Step/>
            <Step/>
            <Step/>
            <Step/>
            <Step/>
        </Steps>
    );
};

export default Stepper;
