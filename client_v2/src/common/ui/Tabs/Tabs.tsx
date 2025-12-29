import React, { ReactNode, useState } from 'react';
import cn from 'clsx';

import { Icon } from '../Icon/Icon';
import { IconType } from '../Icons';
import s from './Tabs.module.pcss';

type TabItem = {
    id: string;
    label: string;
    content: ReactNode;
    icon?: IconType;
};

type Props = {
    tabs: TabItem[];
    defaultActiveTab?: string;
    activeTab?: string;
    onTabChange?: (tabId: string) => void;
    className?: string;
};

export const Tabs = ({ tabs, defaultActiveTab, activeTab: controlledActiveTab, onTabChange, className }: Props) => {
    const [internalActiveTab, setInternalActiveTab] = useState(defaultActiveTab || tabs[0]?.id || '');

    const activeTab = controlledActiveTab !== undefined ? controlledActiveTab : internalActiveTab;

    const handleTabClick = (tabId: string) => {
        if (controlledActiveTab === undefined) {
            setInternalActiveTab(tabId);
        }
        if (onTabChange) {
            onTabChange(tabId);
        }
    };

    const activeTabContent = tabs.find((tab) => tab.id === activeTab)?.content;

    return (
        <div className={cn(s.tabs, className)}>
            <div className={s.nav}>
                {tabs.map((tab) => (
                    <button
                        key={tab.id}
                        type="button"
                        className={cn(s.button, {
                            [s.button_active]: activeTab === tab.id,
                        })}
                        onClick={() => handleTabClick(tab.id)}>
                        {tab.icon && <Icon icon={tab.icon} className={s.icon} />}
                        {tab.label}
                    </button>
                ))}
            </div>
            <div className={s.content}>{activeTabContent}</div>
        </div>
    );
};
