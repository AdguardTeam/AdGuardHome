import cn from 'clsx';
import { Icon, IconType } from 'panel/common/ui/Icon';

import s from './Loader.module.pcss';

type Props = {
    class?: string;
    icon?: IconType;
};

export const InlineLoader = (props: Props) => (
    <Icon class={cn(s.loader, props.class)} icon={props.icon || 'loader'} />
);
