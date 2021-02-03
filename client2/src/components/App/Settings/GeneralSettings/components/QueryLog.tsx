import React, { FC, useContext, useState } from 'react';
import { Radio, Button, Switch } from 'antd';
import { Formik, FormikHelpers } from 'formik';
import { observer } from 'mobx-react-lite';

import { notifySuccess, ConfirmModalLayout } from 'Common/ui';
import { IQueryLogConfig } from 'Entities/QueryLogConfig';
import Store from 'Store';

import { s } from '.';

const { Group } = Radio;

const QueryLog: FC = observer(() => {
    const store = useContext(Store);
    const [showConfirm, setShowConfirm] = useState(false);
    const { ui: { intl }, generalSettings } = store;
    const {
        queryLogConfig,
    } = generalSettings;

    const onSubmit = async (values: IQueryLogConfig, helpers: FormikHelpers<IQueryLogConfig>) => {
        await generalSettings.updateQueryLogConfig(values);
        helpers.setSubmitting(false);
    };

    const onReset = async () => {
        const result = await generalSettings.querylogClear();
        if (result) {
            notifySuccess(intl.getMessage('query_log_cleared'));
        }
    };

    return (
        <>
            <div className={s.title}>
                {intl.getMessage('query_log_configuration')}
                <Button onClick={() => setShowConfirm(true)}>
                    {intl.getMessage('query_log_clear')}
                </Button>
            </div>
            <ConfirmModalLayout
                visible={showConfirm}
                onConfirm={onReset}
                onClose={() => setShowConfirm(false)}
                title={intl.getMessage('query_log_clear')}
                buttonText={intl.getMessage('query_log_clear')}
            >
                {intl.getMessage('query_log_confirm_clear')}
            </ConfirmModalLayout>
            <Formik
                enableReinitialize
                initialValues={queryLogConfig!.serialize()}
                onSubmit={onSubmit}
            >
                {({
                    handleSubmit,
                    values,
                    setFieldValue,
                    isSubmitting,
                    dirty,
                }) => (
                    <form onSubmit={handleSubmit} noValidate className={s.form}>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('query_log_enable')}
                                </div>
                            </div>
                            <Switch checked={values.enabled} onChange={(e) => setFieldValue('enabled', e)}/>
                        </div>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('anonymize_client_ip')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('anonymize_client_ip_desc')}
                                </div>
                            </div>
                            <Switch checked={values.anonymize_client_ip} onChange={(e) => setFieldValue('anonymize_client_ip', e)}/>
                        </div>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('query_log_retention')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('query_log_retention_confirm')}
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

export default QueryLog;
