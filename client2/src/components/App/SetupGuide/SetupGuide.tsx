import React, { FC, useContext } from 'react';
import { Tabs, Grid } from 'antd';

import { InnerLayout } from 'Common/ui';
import { externalLink, p } from 'Common/formating';
import { DHCP_LINK } from 'Consts/common';
import Store from 'Store';

import s from './SetupGuide.module.pcss';

const { useBreakpoint } = Grid;
const { TabPane } = Tabs;

const SetupGuide: FC = () => {
    const store = useContext(Store);
    const { ui: { intl }, system } = store;
    const screens = useBreakpoint();
    const tabsPosition = screens.lg ? 'left' : 'top';

    const { status } = system;

    const tabs = [
        {
            key: intl.getMessage('router'),
            text: intl.getMessage('install_configure_router', { p }),
        },
        {
            key: 'Windows',
            text: intl.getMessage('install_configure_windows', { p }),
        },
        {
            key: 'macOS',
            text: intl.getMessage('install_configure_macos', { p }),
        },
        {
            key: 'Linux',
            text: intl.getMessage('install_configure_router', { p }),
        },
        {
            key: 'Android',
            text: intl.getMessage('install_configure_android', { p }),
        },
        {
            key: 'iOS',
            text: intl.getMessage('install_configure_ios', { p }),
        },
    ];

    const addresses = (
        <>
            <div className={s.text}>
                {intl.getMessage('install_configure_adresses')}
                {status?.dnsAddresses && (
                    <div className={s.addresses}>
                        {status.dnsAddresses.map((address) => (
                            <div className={s.address} key={address}>
                                {address}
                            </div>
                        ))}
                    </div>
                )}
            </div>
            <div className={s.text}>
                {intl.getMessage('install_configure_dhcp', { dhcp: externalLink(DHCP_LINK) })}
            </div>
        </>
    );

    return (
        <InnerLayout title={intl.getMessage('setup_guide')}>
            <Tabs
                defaultActiveKey={intl.getMessage('router')}
                tabPosition={tabsPosition}
                className="tabs"
            >
                {tabs.map((tab) => (
                    <TabPane tab={tab.key} key={tab.key}>
                        <div className={s.title}>
                            {intl.getMessage('install_configure_how_to_title', { value: tab.key })}
                        </div>
                        <div className={s.text}>
                            {tab.text}
                        </div>
                        {addresses}
                    </TabPane>
                ))}
            </Tabs>
        </InnerLayout>
    );
};

export default SetupGuide;
