import { createSignal } from 'solid-js';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { Menu } from 'panel/common/ui/Menu';
import { RoutePath } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './Header.module.pcss';
import { Logo } from '../Sidebar/Logo';

const BURGER_MENU_ID = 'linksMenu';

export const Header = () => {
    const [burgerModal, setBurgerModal] = createSignal(false);
    const [accountSubMenu, setAccountSubMenu] = createSignal(false);

    const closeBurgerMenu = (event: any) => {
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
        <div class={s.header} id="header">
            <div class={s.container}>
                <div class={s.logoWrap}>
                    <div class={theme.layout.mobileOnlyWrapper}>
                        <Icon onClick={openBurgerMenu} class={s.burgerIcon} icon="butter" />
                    </div>
                    <Link to={RoutePath.Dashboard} class={s.link}>
                        <div class={s.linkWrapper}>
                            <Logo id="header" />
                        </div>
                    </Link>
                </div>
            </div>
            <div
                class={cn(s.burgerMenuMask, { [s.open]: burgerModal() })}
                onClick={(event: any) => closeBurgerMenu(event)}
            >
                <div class={s.burgerMenu}>
                    <Menu
                        headerMenu
                        accountSubMenu={accountSubMenu()}
                        setAccountSubMenu={setAccountSubMenu}
                        burgerMenuId={BURGER_MENU_ID}
                        closeSubMenu={closeSubMenu}
                    />
                </div>
            </div>
        </div>
    );
};
