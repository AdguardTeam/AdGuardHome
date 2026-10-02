import cn from 'clsx';

import { TruncatedText } from 'panel/common/ui/TruncatedText';
import theme from 'panel/lib/theme';

import s from './NameCell.module.pcss';

type Props = {
    name: string;
    testId?: string;
};

export const NameCell = (props: Props) => (
    <TruncatedText
        text={props.name}
        testId={props.testId}
        class={cn(theme.text.t3, theme.text.condenced, s.nameCell)}
    />
);
