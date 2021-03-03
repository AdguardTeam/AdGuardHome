import React, { FC, useContext, useState } from 'react';
import { Radio, Button } from 'antd';
import { Formik, FormikHelpers } from 'formik';
import { observer } from 'mobx-react-lite';

import { notifySuccess, ConfirmModalLayout } from 'Common/ui';
import { IStatsConfig } from 'Entities/StatsConfig';
import Store from 'Store';

import { s } from '.';

const { Group } = Radio;

const Statistics: FC = observer(() => {
    const store = useContext(Store);
    const [showConfirm, setShowConfirm] = useState(false);
    const { ui: { intl }, generalSettings } = store;
    const {
        statsConfig,
    } = generalSettings;

    const onSubmit = async (values: IStatsConfig, helpers: FormikHelpers<IStatsConfig>) => {
        await generalSettings.updateStatsConfig(values);
        helpers.setSubmitting(false);
    };

    const onReset = async () => {
        const result = await generalSettings.statsReset();
        if (result) {
            notifySuccess(intl.getMessage('stats_reset'));
        }
    };

    return (
        <>
            <div className={s.title}>
                {intl.getMessage('statistics_configuration')}
                <Button onClick={() => setShowConfirm(true)}>
                    {intl.getMessage('statistics_clear')}
                </Button>
            </div>
            <ConfirmModalLayout
                visible={showConfirm}
                onConfirm={onReset}
                onClose={() => setShowConfirm(false)}
                title={intl.getMessage('statistics_clear')}
                buttonText={intl.getMessage('statistics_clear')}
            >
                {intl.getMessage('statistics_clear_confirm')}
            </ConfirmModalLayout>
            <Formik
                enableReinitialize
                initialValues={statsConfig!.serialize()}
                onSubmit={onSubmit}
            >
                {({
                    handleSubmit,
                    values,
                    setFieldValue,
                    isSubmitting,
                    dirty,
                }) => (
                    <form onSubmit={handleSubmit} noValidate>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('statistics_retention')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('statistics_retention_desc')}
                                </div>
                            </div>
                        </div>
                        <Group value={values.interval} onChange={(e) => setFieldValue('interval', e.target.value)}>
                            <Radio value={1} className={s.radio}>
                                {intl.getMessage('interval_24_hour')}
                            </Radio>
                            <Radio value={7} className={s.radio}>
                                {intl.getPlural('interval_days', 7, { count: 7 })}
                            </Radio>
                            <Radio value={30} className={s.radio}>
                                {intl.getPlural('interval_days', 30, { count: 30 })}
                            </Radio>
                            <Radio value={90} className={s.radio}>
                                {intl.getPlural('interval_days', 90, { count: 90 })}
                            </Radio>
                        </Group>
                        {dirty && (
                            <Button
                                type="primary"
                                htmlType="submit"
                                className={s.save}
                                disabled={isSubmitting}
                            >
                                {intl.getMessage('save_btn')}
                            </Button>
                        )}
                    </form>
                )}
            </Formik>
        </>
    );
});

export default Statistics;
