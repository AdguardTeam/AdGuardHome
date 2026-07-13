import { createSignal, createEffect, Show, For } from 'solid-js';
import cn from 'clsx';
import { Icon, type IconType } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { type RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type AccordionItem = {
    label: string;
    path: string;
    routePath: RoutePathKey;
};

type Props = {
    title: string;
    icon: IconType;
    items: AccordionItem[];
    isActive: (path: string | string[], full?: boolean) => boolean;
    class?: string;
};

export const AccordionSection = (props: Props) => {
    const [isOpen, setIsOpen] = createSignal(false);

    const isAnyItemActive = () => props.items.some((item) => props.isActive(item.path));

    createEffect(() => {
        if (!isOpen() && isAnyItemActive()) {
            setIsOpen(true);
        }
    });

    const toggleAccordion = (e: MouseEvent) => {
        e.stopPropagation();
        setIsOpen(!isOpen());
    };

    return (
        <div class={cn(s.menuLinkWrapper, props.class)}>
            <div
                class={cn(s.menuLink, s.accordionHeader, {
                    [s.activeLink]: isAnyItemActive(),
                    [s.accordionOpen]: isOpen(),
                })}
                onClick={toggleAccordion}
                role="button"
                tabIndex={0}
            >
                <Icon class={s.linkIcon} icon={props.icon} />
                <span class={theme.common.textOverflow}>{props.title}</span>
                <Icon
                    class={cn(s.accordionArrow, { [s.accordionArrowOpen]: isOpen() })}
                    icon="arrow_bottom"
                />
            </div>
            <Show when={isOpen()}>
                <div class={s.accordionContent}>
                    <For each={props.items}>
                        {(item) => (
                            <div class={s.accordionItem}>
                                <Link
                                    class={cn(s.accordionLink, {
                                        [s.activeLink]: props.isActive(item.path),
                                    })}
                                    to={item.routePath}
                                >
                                    <span class={theme.common.textOverflow}>{item.label}</span>
                                </Link>
                            </div>
                        )}
                    </For>
                </div>
            </Show>
        </div>
    );
};
