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
};

export const PlusButton = (props: Props) => (
    <button
        type="button"
        class={cn(s.plusButton, props.class)}
        onClick={props.onClick}
        disabled={props.disabled}
        data-testid={props.testId}
    >
        <Icon icon="plus" />
        {props.children}
    </button>
);
