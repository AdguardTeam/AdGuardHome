import React, { MouseEvent, useState } from 'react';
import cn from 'clsx';
import { Icon, Link, Menu } from 'panel/common/ui';
import { RoutePath } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './Header.module.pcss';
import { Logo } from '../Sidebar/Logo';

const BURGER_MENU_ID = 'linksMenu';

export const Header = () => {
    const [burgerModal, setBurgerModal] = useState(false);
    const [accountSubMenu, setAccountSubMenu] = useState(false);

    const closeBurgerMenu = (event: MouseEvent<HTMLDivElement>) => {
        const target = event.target as HTMLDivElement;

        if (!target.closest(`#${BURGER_MENU_ID}`)) {
            setBurgerModal(false);
            document.body.classList.remove('block-scroll');
        }
    };

    const closeSubMenu = () => {
        document.body.classList.remove('block-scroll');
        setBurgerModal(false);
        setAccountSubMenu(false);
    };

    const openBurgerMenu = () => {
        document.body.classList.add('block-scroll');
        setBurgerModal(true);
    };

    return (
        <div className={s.header} id="header">
            <div className={s.container}>
                <div className={s.logoWrap}>
                    <div className={theme.layout.mobileOnlyWrapper}>
                        <Icon onClick={openBurgerMenu} className={s.burgerIcon} icon="butter" />
                    </div>
                    <Link to={RoutePath.Dashboard} className={s.link}>
                        <div className={s.linkWrapper}>
                            <Logo id="header" />
                        </div>
                    </Link>
                </div>
            </div>
            <div
                className={cn(s.burgerMenuMask, { [s.open]: burgerModal })}
                onClick={(event) => closeBurgerMenu(event)}>
                <div className={s.burgerMenu}>
                    <Menu
                        headerMenu
                        accountSubMenu={accountSubMenu}
                        setAccountSubMenu={setAccountSubMenu}
                        burgerMenuId={BURGER_MENU_ID}
                        closeSubMenu={closeSubMenu}
                    />
                </div>
            </div>
        </div>
    );
};
