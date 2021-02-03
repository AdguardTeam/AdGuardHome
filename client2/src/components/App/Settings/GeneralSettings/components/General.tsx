import React, { FC, useContext } from 'react';
import { Button, Switch, Select } from 'antd';
import { Formik, FormikHelpers } from 'formik';
import { observer } from 'mobx-react-lite';

import { Link } from 'Common/ui';
import Store from 'Store';
import { RoutePath } from 'Paths';

import { s } from '.';

const { Option } = Select;

const General: FC = observer(() => {
    const store = useContext(Store);
    const { ui: { intl }, generalSettings } = store;
    const {
        safebrowsing,
        filteringConfig,
        parental,
        safesearch,
    } = generalSettings;

    const initialValues = {
        ...filteringConfig!.serialize(),
        safebrowsing,
        parental,
        safesearch,
    };

    type InitialValues = typeof initialValues;

    const onSubmit = async (values: InitialValues, helpers: FormikHelpers<InitialValues>) => {
        // await generalSettings.updateQueryLogConfig(values);
        if (initialValues.parental !== values.parental) {
            generalSettings[values.parental ? 'parentalEnable' : 'parentalDisable']();
        }
        if (initialValues.safesearch !== values.safesearch) {
            generalSettings[values.safesearch ? 'safebrowsingEnable' : 'safebrowsingDisable']();
        }
        if (initialValues.safebrowsing !== values.safebrowsing) {
            generalSettings[values.safebrowsing ? 'safebrowsingEnable' : 'safebrowsingDisable']();
        }
        if (initialValues.enabled !== values.enabled
            || initialValues.interval !== values.interval) {
            generalSettings.updateFilteringConfig({
                interval: values.interval,
                enabled: values.enabled,
            });
        }
        helpers.setSubmitting(false);
    };

    const filtersLink = (e: string) => {
        // TODO: fix link
        return <Link to={RoutePath.Dashboard}>{e}</Link>;
    };

    return (
        <>
            <div className={s.title}>
                {intl.getMessage('filter_category_general')}
            </div>
            <Formik
                enableReinitialize
                initialValues={initialValues}
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
                                    {intl.getMessage('block_domain_use_filters_and_hosts')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('filters_block_toggle_hint', { a: filtersLink })}
                                </div>
                            </div>
                            <Switch checked={values.enabled} onChange={(e) => setFieldValue('enabled', e)}/>
                        </div>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('filters_interval')}
                                </div>
                            </div>
                        </div>
                        <Select
                            value={values.interval}
                            onChange={(e) => setFieldValue('interval', e)}
                            className={s.select}
                        >
                            <Option value={0}>
                                {intl.getMessage('disabled')}
                            </Option>
                            <Option value={1}>
                                {intl.getPlural('interval_hours', 1, { count: 1 })}
                            </Option>
                            <Option value={12}>
                                {intl.getPlural('interval_hours', 12, { count: 12 })}
                            </Option>
                            <Option value={24}>
                                {intl.getPlural('interval_hours', 24, { count: 24 })}
                            </Option>
                            <Option value={72}>
                                {intl.getPlural('interval_days', 3, { count: 3 })}
                            </Option>
                            <Option value={168}>
                                {intl.getPlural('interval_days', 7, { count: 7 })}
                            </Option>
                        </Select>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('use_adguard_browsing_sec')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('use_adguard_browsing_sec_hint')}
                                </div>
                            </div>
                            <Switch checked={values.safebrowsing} onChange={(e) => setFieldValue('safebrowsing', e)}/>
                        </div>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('use_adguard_parental')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('use_adguard_parental_hint')}
                                </div>
                            </div>
                            <Switch checked={values.parental} onChange={(e) => setFieldValue('parental', e)}/>
                        </div>
                        <div className={s.item}>
                            <div>
                                <div className={s.nameTitle}>
                                    {intl.getMessage('enforce_safe_search')}
                                </div>
                                <div className={s.nameDesc}>
                                    {intl.getMessage('enforce_save_search_hint')}
                                </div>
                            </div>
                            <Switch checked={values.safesearch} onChange={(e) => setFieldValue('safesearch', e)}/>
                        </div>
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

export default General;
