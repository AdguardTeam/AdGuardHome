import { type JSX, createSignal, createMemo, For } from 'solid-js';
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
};

export const Tabs = (props: Props) => {
    const [internalActiveTab, setInternalActiveTab] = createSignal(
        props.defaultActiveTab || props.tabs[0]?.id || '',
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
        <div class={cn(s.tabs, props.class)}>
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
                            {tab.icon && <Icon icon={tab.icon} class={s.icon} />}
                            {tab.label}
                        </button>
                    )}
                </For>
            </div>
            <div class={cn(s.content, props.contentClass)}>{activeTabContent()}</div>
        </div>
    );
};
