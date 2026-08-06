import { type JSX, createSignal, Show } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './Accordion.module.pcss';

type Props = {
    title: JSX.Element;
    children: JSX.Element;
    defaultOpen?: boolean;
    class?: string;
    compact?: boolean;
};

export const Accordion = (props: Props) => {
    const [isOpen, setIsOpen] = createSignal(props.defaultOpen ?? false);

    const toggleOpen = () => {
        setIsOpen(!isOpen());
    };

    return (
        <div class={cn(s.accordion, props.class)}>
            <button
                type="button"
                class={cn(s.header, { [s.compact]: props.compact })}
                onClick={toggleOpen}
                aria-expanded={isOpen()}
                aria-controls="accordion-content"
            >
                <Icon icon="arrow_bottom" class={cn(s.arrow, { [s.arrowOpen]: isOpen() })} />
                <span class={cn(s.title, theme.text.t2, theme.text.semibold)}>{props.title}</span>
            </button>

            <Show when={isOpen()}>
                <div id="accordion-content">{props.children}</div>
            </Show>
        </div>
    );
};
