import React, { useContext, useState } from 'react';
import { Icon, Link } from 'panel/common/ui';
import { RoutePath } from 'panel/components/Routes/Paths';

import s from './styles.module.pcss';
import { Menu } from '../Menu/Menu';
import { Logo } from './Logo';

export const Sidebar = () => {
    const [accountSubMenu, setAccountSubMenu] = useState(false);

    return (
        <div className={s.sidebarWrapper} id="menu">
            <div className={s.sidebarContainer}>
                <div className={s.sidebar}>
                    <div className={s.container}>
                        <Link to={RoutePath.Dashboard} className={s.link}>
                            <div className={s.linkWrapper}>
                                <Logo id="sidebar" />
                            </div>
                        </Link>
                    </div>
                    <Menu rightSideDropdown accountSubMenu={accountSubMenu} setAccountSubMenu={setAccountSubMenu} />
                </div>
            </div>
        </div>
    );
};
