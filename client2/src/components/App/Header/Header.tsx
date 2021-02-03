import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import { MenuOutlined } from '@ant-design/icons';
import { observer } from 'mobx-react-lite';

import { Icon, LangSelect } from 'Common/ui';
import Store from 'Store';

import s from './Header.module.pcss';

const Header: FC = observer(() => {
    const store = useContext(Store);
    const { ui: { intl }, system, ui } = store;
    const { status, profile } = system;

    const updateServerStatus = () => {
        system.switchServerStatus(!status?.protectionEnabled);
    };

    return (
        <div className={s.header}>
            <div className={s.top}>
                <Button
                    icon={<MenuOutlined />}
                    className={s.menu}
                    onClick={() => ui.toggleSidebar()}
                />
            </div>
            <div className={s.bottom}>
                <div className={s.status}>
                    <Icon icon="logo_shield" className={s.icon} />
                    {status?.protectionEnabled
                        ? intl.getMessage('header_adguard_status_enabled')
                        : intl.getMessage('header_adguard_status_disabled')}
                </div>
                <Button
                    type="ghost"
                    size="small"
                    className={s.action}
                    onClick={updateServerStatus}
                >
                    {status?.protectionEnabled
                        ? intl.getMessage('disable')
                        : intl.getMessage('enable')}
                </Button>
                {profile?.name && (
                    <div className={s.user}>
                        <Icon icon="user" className={s.icon} />
                        {profile?.name}
                    </div>
                )}
                <div className={s.languages}>
                    <LangSelect />
                </div>
            </div>
        </div>
    );
});

export default Header;
