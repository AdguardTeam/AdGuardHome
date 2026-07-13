import { createSignal } from 'solid-js';
import { Link } from 'panel/common/ui/Link';
import { Menu } from 'panel/common/ui/Menu';
import { RoutePath } from 'panel/components/Routes/Paths';

import s from './styles.module.pcss';
import { Logo } from './Logo';

export const Sidebar = () => {
    const [accountSubMenu, setAccountSubMenu] = createSignal(false);

    return (
        <div class={s.sidebarWrapper} id="menu">
            <div class={s.sidebarContainer}>
                <div class={s.sidebar}>
                    <div class={s.container}>
                        <Link to={RoutePath.Dashboard} class={s.link}>
                            <div class={s.linkWrapper}>
                                <Logo id="sidebar" />
                            </div>
                        </Link>
                    </div>
                    <Menu
                        rightSideDropdown
                        accountSubMenu={accountSubMenu()}
                        setAccountSubMenu={setAccountSubMenu}
                    />
                </div>
            </div>
        </div>
    );
};
