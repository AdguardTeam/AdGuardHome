import React, { FC, useContext } from 'react';
import { Layout, Menu, Grid } from 'antd';
import { observer } from 'mobx-react-lite';
import { PieChartOutlined, FormOutlined, TableOutlined, ProfileOutlined, SettingOutlined } from '@ant-design/icons';

import Store from 'Store';
import { Link, Icon, Mask } from 'Common/ui';
import { RoutePath, linkPathBuilder } from 'Components/App/Routes/Paths';

import s from './Sidebar.module.pcss';

const { Sider } = Layout;
const { Item: MenuItem, SubMenu } = Menu;
const { useBreakpoint } = Grid;

const Sidebar: FC = observer(() => {
    const store = useContext(Store);
    const screens = useBreakpoint();
    const { ui: { intl, sidebarOpen, toggleSidebar } } = store;

    if (!Object.keys(screens).length) {
        return null;
    }

    const handleSidebar = () => {
        if (!screens.xl) {
            toggleSidebar();
        }
    };

    return (
        <>
            <Sider
                collapsed={!sidebarOpen && !screens.xl}
                collapsedWidth={0}
                collapsible
                onClick={handleSidebar}
                className="sidebar"
                trigger={null}
                width={200}
            >
                <Icon icon="logo_light" className={s.logo} />
                <Menu
                    mode="inline"
                    theme="dark"
                    className={s.menu}
                >
                    <MenuItem key={linkPathBuilder(RoutePath.Dashboard)}>
                        <Link to={RoutePath.Dashboard}>
                            <PieChartOutlined className={s.icon} />
                            {intl.getMessage('dashboard')}
                        </Link>
                    </MenuItem>
                    <MenuItem key={linkPathBuilder(RoutePath.FiltersBlocklist)}>
                        <Link to={RoutePath.FiltersBlocklist}>
                            <FormOutlined className={s.icon} />
                            {intl.getMessage('filters')}
                        </Link>
                    </MenuItem>
                    <MenuItem key={linkPathBuilder(RoutePath.QueryLog)}>
                        <Link to={RoutePath.QueryLog}>
                            <TableOutlined className={s.icon} />
                            {intl.getMessage('query_log')}
                        </Link>
                    </MenuItem>
                    <MenuItem key={linkPathBuilder(RoutePath.SetupGuide)}>
                        <Link to={RoutePath.SetupGuide}>
                            <ProfileOutlined className={s.icon} />
                            {intl.getMessage('setup_guide')}
                        </Link>
                    </MenuItem>
                    <SubMenu
                        key="settings"
                        icon={<SettingOutlined className={s.icon} />}
                        title={intl.getMessage('settings')}
                    >
                        <Menu.Item key={linkPathBuilder(RoutePath.SettingsGeneral)}>
                            <Link to={RoutePath.SettingsGeneral}>
                                {intl.getMessage('general_settings')}
                            </Link>
                        </Menu.Item>
                        <Menu.Item key={linkPathBuilder(RoutePath.SettingsDns)}>
                            <Link to={RoutePath.SettingsDns}>
                                {intl.getMessage('dns_settings')}
                            </Link>
                        </Menu.Item>
                        <Menu.Item key={linkPathBuilder(RoutePath.SettingsEncryption)}>
                            <Link to={RoutePath.SettingsEncryption}>
                                {intl.getMessage('encryption_settings')}
                            </Link>
                        </Menu.Item>
                        <Menu.Item key={linkPathBuilder(RoutePath.SettingsClients)}>
                            <Link to={RoutePath.SettingsClients}>
                                {intl.getMessage('client_settings')}
                            </Link>
                        </Menu.Item>
                        <Menu.Item key={linkPathBuilder(RoutePath.SettingsDhcp)}>
                            <Link to={RoutePath.SettingsDhcp}>
                                {intl.getMessage('dhcp_settings')}
                            </Link>
                        </Menu.Item>
                    </SubMenu>
                    <MenuItem className={s.logout}>
                        <a href="control/logout">
                            <Icon icon="sign_out" className={s.icon} />
                            {intl.getMessage('sign_out')}
                        </a>
                    </MenuItem>
                </Menu>
            </Sider>
            <Mask open={sidebarOpen} handle={handleSidebar} />
        </>
    );
});

export default Sidebar;
