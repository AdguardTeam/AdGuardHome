import React, { FC, useContext, useEffect } from 'react';
import { Tabs, Grid } from 'antd';
import { observer } from 'mobx-react-lite';

import { InnerLayout } from 'Common/ui';
import Store from 'Store';

import { General, QueryLog, Statistics, TAB_KEY } from './components';

const { useBreakpoint } = Grid;
const { TabPane } = Tabs;

const GeneralSettings: FC = observer(() => {
    const store = useContext(Store);
    const { ui: { intl }, generalSettings } = store;
    const { inited } = generalSettings;
    const screens = useBreakpoint();

    useEffect(() => {
        if (!inited) {
            generalSettings.init();
        }
    }, [inited]);

    if (!inited) {
        return null;
    }

    const tabsPosition = screens.lg ? 'left' : 'top';

    return (
        <InnerLayout title={intl.getMessage('general_settings')}>
            <Tabs
                defaultActiveKey={TAB_KEY.GENERAL}
                tabPosition={tabsPosition}
                className="tabs"
            >
                <TabPane tab={intl.getMessage('filter_category_general')} key={TAB_KEY.GENERAL}>
                    <General/>
                </TabPane>
                <TabPane tab={intl.getMessage('query_log_configuration')} key={TAB_KEY.QUERY_LOG}>
                    <QueryLog/>
                </TabPane>
                <TabPane tab={intl.getMessage('statistics_configuration')} key={TAB_KEY.STATISTICS}>
                    <Statistics/>
                </TabPane>
            </Tabs>
        </InnerLayout>
    );
});

export default GeneralSettings;
