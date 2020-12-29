import React, { FC, useContext } from 'react';
import { Tabs } from 'antd';
import cn from 'classnames';
import { FormikHelpers } from 'formik';

import Store from 'Store/installStore';
import theme from 'Lib/theme';
import { danger, p } from 'Common/formating';
import { DEFAULT_DNS_PORT, DEFAULT_IP_ADDRESS, DEFAULT_IP_PORT } from 'Consts/install';

import { FormValues } from '../../Install';
import StepButtons from '../StepButtons';
import s from './ConfigureDevices.module.pcss';

const { TabPane } = Tabs;

interface ConfigureDevicesProps {
    values: FormValues;
    setFieldValue: FormikHelpers<FormValues>['setFieldValue'];
}

const ConfigureDevices: FC<ConfigureDevicesProps> = ({
    values, setFieldValue,
}) => {
    const { ui: { intl }, install: { addresses } } = useContext(Store);

    const dhcp = (e: string) => (
        // TODO: link to dhcp
        <a href="http://" target="_blank" rel="noopener noreferrer">{e}</a>
    );

    const allIps = addresses?.interfaces.reduce<string[]>((all, data) => {
        const { ipAddresses } = data;
        if (ipAddresses) {
            all.push(...ipAddresses);
        }
        return all;
    }, [] as string[]);

    const { web: { ip: webIp }, dns: { ip: dnsIp } } = values;
    const selectedWebIps = webIp.length === 1 && webIp[0] === DEFAULT_IP_ADDRESS
        ? allIps : webIp;
    const selectedDnsIps = dnsIp.length === 1 && dnsIp[0] === DEFAULT_IP_ADDRESS
        ? allIps : dnsIp;

    return (
        <div>
            <div className={theme.typography.title}>
                {intl.getMessage('install_configure_title')}
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_block)}>
                {intl.getMessage('install_configure_danger_notice', { danger })}
            </div>
            <div className={theme.typography.subTitle}>
                {intl.getMessage('install_configure_how_to_title')}
            </div>
            <Tabs defaultActiveKey="1" tabPosition="left" className={s.tabs}>
                <TabPane tab="Router" key="1">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {intl.getMessage('install_configure_router', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Windows" key="2">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {intl.getMessage('install_configure_windows', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Macos" key="3">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {intl.getMessage('install_configure_macos', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Linux" key="4">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {/* TODO: add linux setup */}
                        {intl.getMessage('install_configure_router', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Android" key="5">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {intl.getMessage('install_configure_android', { p })}
                    </div>
                </TabPane>
                <TabPane tab="iOs" key="6">
                    <div className={cn(theme.typography.text, theme.typography.text_base)}>
                        {intl.getMessage('install_configure_ios', { p })}
                    </div>
                </TabPane>
            </Tabs>

            <div className={theme.typography.subTitle}>
                {intl.getMessage('install_configure_adresses')}
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_base)}>
                <p>
                    {intl.getMessage('install_admin_interface_title')}
                </p>
                <p>
                    {selectedWebIps?.map((ip) => (
                        <div key={ip}>
                            {ip}{values.web.port !== DEFAULT_IP_PORT && `:${values.web.port}`}
                        </div>
                    ))}
                </p>
                <p>
                    {intl.getMessage('install_dns_server_title')}
                </p>
                <div>
                    {selectedDnsIps?.map((ip) => (
                        <div key={ip}>
                            {ip}{values.dns.port !== DEFAULT_DNS_PORT && `:${values.dns.port}`}
                        </div>
                    ))}
                </div>
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_base)}>
                {intl.getMessage('install_configure_dhcp', { dhcp })}
            </div>
            <StepButtons
                setFieldValue={setFieldValue}
                currentStep={4}
                values={values}
            />
        </div>
    );
};

export default ConfigureDevices;
