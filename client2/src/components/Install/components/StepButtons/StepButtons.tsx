import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import { observer } from 'mobx-react-lite';
import { FormikHelpers } from 'formik';

import Store from 'Store/installStore';
import theme from 'Lib/theme';

import { FormValues } from '../../Install';

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
        <div className={theme.install.actions}>
            <Button
                size="large"
                type="ghost"
                className={theme.install.button}
                onClick={() => setFieldValue('step', currentStep - 1)}
            >
                {intl.getMessage('back')}
            </Button>
            <Button
                size="large"
                type="primary"
                htmlType="submit"
                className={theme.install.button}
            >
                {intl.getMessage('next')}
            </Button>
        </div>
    );
});

export default StepButtons;
