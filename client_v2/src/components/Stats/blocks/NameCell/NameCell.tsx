import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './NameCell.module.pcss';

type Props = {
    name: string;
    testId?: string;
};

export const NameCell = (props: Props) => (
    <span
        class={cn(theme.text.t3, theme.text.condenced, theme.common.textOverflow, s.nameCell)}
        title={props.name}
        data-testid={props.testId}
    >
        {props.name}
    </span>
);
