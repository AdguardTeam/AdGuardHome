import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Icons } from 'panel/common/ui/Icons';
import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { Toasts } from 'panel/components/Toasts';
import theme from 'panel/lib/theme';

import styles from './AuthLayout.module.pcss';

type Props = {
    title: string;
    children: JSX.Element;
};

export const AuthLayout = (props: Props) => (
    <div class={styles.wrapper}>
        <PublicHeader variant="auth" dropdownPosition="bottomRight" useLocalLanguage={true} />

        <main class={styles.content}>
            <div class={styles.card}>
                <h1 class={cn(styles.title, theme.title.h4, theme.title.h3_tablet)}>
                    {props.title}
                </h1>
                {props.children}
            </div>
        </main>

        <Toasts />

        <Icons />
    </div>
);
