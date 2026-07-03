import { type JSX, createSignal, createMemo, For, Show, untrack } from 'solid-js';
import cn from 'clsx';

import { Icon } from '../Icon/Icon';
import { IconType } from '../Icons';
import s from './Tabs.module.pcss';

type TabItem = {
    id: string;
    label: string;
    content: JSX.Element;
    icon?: IconType;
};

type Props = {
    tabs: TabItem[];
    defaultActiveTab?: string;
    activeTab?: string;
    onTabChange?: (tabId: string) => void;
    class?: string;
    contentClass?: string;
    fullWidth?: boolean;
    variant?: 'default' | 'filled';
};

export const Tabs = (props: Props) => {
    const [internalActiveTab, setInternalActiveTab] = createSignal(
        untrack(() => props.defaultActiveTab || props.tabs[0]?.id || ''),
    );

    const activeTab = createMemo(() =>
        props.activeTab !== undefined ? props.activeTab : internalActiveTab(),
    );

    const handleTabClick = (tabId: string) => {
        if (props.activeTab === undefined) {
            setInternalActiveTab(tabId);
        }
        props.onTabChange?.(tabId);
    };

    const activeTabContent = createMemo(
        () => props.tabs.find((tab) => tab.id === activeTab())?.content,
    );

    return (
        <div
            class={cn(s.tabs, props.class, {
                [s.tabs_filled]: props.variant === 'filled',
            })}
        >
            <div class={s.nav}>
                <For each={props.tabs}>
                    {(tab) => (
                        <button
                            type="button"
                            class={cn(s.button, {
                                [s.button_active]: activeTab() === tab.id,
                            })}
                            onClick={() => handleTabClick(tab.id)}
                        >
                            <Show when={tab.icon}>
                                <Icon icon={tab.icon} class={s.icon} />
                            </Show>
                            {tab.label}
                        </button>
                    )}
                </For>
            </div>
            <div
                class={cn(s.content, props.contentClass, {
                    [s.content_fullWidth]: props.fullWidth,
                })}
            >
                {activeTabContent()}
            </div>
        </div>
    );
};
