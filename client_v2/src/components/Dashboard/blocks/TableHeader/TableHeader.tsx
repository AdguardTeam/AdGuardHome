import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './TableHeader.module.pcss';

type Props = {
    nameLabel: string;
    countLabel: string;
};

export const TableHeader = (props: Props) => (
    <div class={cn(theme.text.t3, theme.text.semibold, s.tableHeader)}>
        <span>{props.nameLabel}</span>
        <span>{props.countLabel}</span>
    </div>
);
