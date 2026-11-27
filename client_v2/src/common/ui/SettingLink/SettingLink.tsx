import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import type { LinkParams, RoutePathKey } from 'panel/components/Routes/Paths';

import s from './SettingLink.module.pcss';

type Props = {
    to: RoutePathKey;
    linkProps?: LinkParams;
    title: string;
    description: string;
    disabled?: boolean;
    class?: string;
    'data-testid'?: string;
};

/**
 * A navigational settings row: title, description, and a trailing chevron.
 *
 * Unlike `SettingRow`'s `link` variant, which is a button-like row that runs
 * an `onClick`, this renders a real router link, so the row keeps anchor
 * semantics (href, middle-click, open in a new tab) while sharing the
 * disabled styling with the other client sub-page rows.
 */
export const SettingLink = (props: Props) => (
    <Link
        to={props.to}
        props={props.linkProps}
        disabled={props.disabled}
        class={cn(s.link, { [s.linkDisabled]: props.disabled }, props.class)}
        data-testid={props['data-testid']}
    >
        <div class={s.content}>
            <div class={s.title}>{props.title}</div>
            <div class={s.description}>{props.description}</div>
        </div>
        <Icon icon="arrow" color="gray" class={s.arrow} />
    </Link>
);
