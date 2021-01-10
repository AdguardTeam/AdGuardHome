import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import cn from 'classnames';
import { observer } from 'mobx-react-lite';
import { FormikHelpers } from 'formik';

import Store from 'Store/installStore';

import { FormValues } from '../../Install';
import s from './StepButtons.module.pcss';

interface StepButtonsProps {
    setFieldValue: FormikHelpers<FormValues>['setFieldValue'];
    currentStep: number;
    values: FormValues;
}

const StepButtons: FC<StepButtonsProps> = observer(({
    setFieldValue,
    currentStep,
}) => {
    const { ui: { intl } } = useContext(Store);
    return (
        <div>
            <Button
                size="large"
                type="ghost"
                className={cn(s.button, s.inGroup)}
                onClick={() => setFieldValue('step', currentStep - 1)}
            >
                {intl.getMessage('back')}
            </Button>
            <Button
                size="large"
                type="primary"
                htmlType="submit"
                className={cn(s.button)}
            >
                {intl.getMessage('next')}
            </Button>
        </div>
    );
});

export default StepButtons;
