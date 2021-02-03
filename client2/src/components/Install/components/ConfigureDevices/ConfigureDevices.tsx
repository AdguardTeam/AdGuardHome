import React, { FC, useContext } from 'react';
import { Tabs, Grid } from 'antd';
import cn from 'classnames';
import { FormikHelpers } from 'formik';

import { DHCP_LINK } from 'Consts/common';
import { danger, externalLink, p } from 'Common/formating';
import { DEFAULT_DNS_PORT, DEFAULT_IP_ADDRESS, DEFAULT_IP_PORT } from 'Consts/install';
import Store from 'Store/installStore';
import theme from 'Lib/theme';

import { FormValues } from '../../Install';
import StepButtons from '../StepButtons';

const { useBreakpoint } = Grid;
const { TabPane } = Tabs;

interface ConfigureDevicesProps {
    values: FormValues;
    setFieldValue: FormikHelpers<FormValues>['setFieldValue'];
}

const ConfigureDevices: FC<ConfigureDevicesProps> = ({
    values, setFieldValue,
}) => {
    const { ui: { intl }, install: { addresses } } = useContext(Store);
    const screens = useBreakpoint();
    const tabsPosition = screens.md ? 'left' : 'top';

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
        <>
            <div className={theme.install.title}>
                {intl.getMessage('install_configure_title')}
            </div>
            <div className={cn(theme.install.text, theme.install.text_block)}>
                {intl.getMessage('install_configure_danger_notice', { danger })}
            </div>

            <Tabs defaultActiveKey="1" tabPosition={tabsPosition} className={theme.install.tabs}>
                <TabPane tab={intl.getMessage('router')} key="1">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: intl.getMessage('router') })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {intl.getMessage('install_configure_router', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Windows" key="2">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: 'Windows' })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {intl.getMessage('install_configure_windows', { p })}
                    </div>
                </TabPane>
                <TabPane tab="macOS" key="3">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: 'macOS' })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {intl.getMessage('install_configure_macos', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Linux" key="4">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: 'Linux' })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {/* TODO: add linux setup */}
                        {intl.getMessage('install_configure_router', { p })}
                    </div>
                </TabPane>
                <TabPane tab="Android" key="5">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: 'Android' })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {intl.getMessage('install_configure_android', { p })}
                    </div>
                </TabPane>
                <TabPane tab="iOS" key="6">
                    <div className={theme.install.subtitle}>
                        {intl.getMessage('install_configure_how_to_title', { value: 'iOS' })}
                    </div>
                    <div className={cn(theme.install.text, theme.install.text_base)}>
                        {intl.getMessage('install_configure_ios', { p })}
                    </div>
                </TabPane>
            </Tabs>

            <div className={theme.install.subtitle}>
                {intl.getMessage('install_configure_adresses')}
            </div>
            <div className={cn(theme.install.text, theme.install.text_block)}>
                <div className={cn(theme.install.text, theme.install.text_base)}>
                    {intl.getMessage('install_admin_interface_title')}
                </div>
                <div className={cn(theme.install.text, theme.install.text_base)}>
                    {selectedWebIps?.map((ip) => (
                        <div key={ip} className={theme.install.ip}>
                            {ip}{values.web.port !== DEFAULT_IP_PORT && `:${values.web.port}`}
                        </div>
                    ))}
                </div>
                <div className={cn(theme.install.text, theme.install.text_base)}>
                    {intl.getMessage('install_dns_server_title')}
                </div>
                <div className={cn(theme.install.text, theme.install.text_base)}>
                    {selectedDnsIps?.map((ip) => (
                        <div key={ip} className={theme.install.ip}>
                            {ip}{values.dns.port !== DEFAULT_DNS_PORT && `:${values.dns.port}`}
                        </div>
                    ))}
                </div>
            </div>
            <div className={cn(theme.install.text, theme.install.text_base)}>
                {intl.getMessage('install_configure_dhcp', { dhcp: externalLink(DHCP_LINK) })}
            </div>
            <StepButtons
                setFieldValue={setFieldValue}
                currentStep={4}
                values={values}
            />
        </>
    );
};

export default ConfigureDevices;
