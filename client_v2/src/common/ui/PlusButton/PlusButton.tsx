import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import s from './PlusButton.module.pcss';

type Props = {
    children: JSX.Element;
    class?: string;
    onClick: () => void;
    disabled?: boolean;
    testId?: string;
    weight?: 'normal' | 'semi' | 'bold';
};

export const PlusButton = (props: Props) => (
    <button
        type="button"
        class={cn(s.plusButton, { [s[props.weight]]: props.weight }, props.class)}
        onClick={(e) => (props.onClick as any)?.(e)}
        disabled={props.disabled}
        data-testid={props.testId}
    >
        <Icon icon="plus" />
        {props.children}
    </button>
);
