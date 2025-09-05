import React from 'react';

import intl from 'panel/common/intl';
import { MODAL_TYPE, TAB_TYPE } from 'panel/helpers/constants';
import { Tabs } from 'panel/common/ui/Tabs';
import { FiltersList } from './FiltersList';
import { ManualFilterForm } from './ManualFilterForm';

type Props = {
    modalType: string;
    selectedSources?: Record<string, boolean>;
    activeTab: string;
    onTabChange: (tabId: string) => void;
};

export const FormContent = ({ modalType, selectedSources, activeTab, onTabChange }: Props) => {
    if (modalType === MODAL_TYPE.SELECT_MODAL_TYPE) {
        return (
            <Tabs
                activeTab={activeTab}
                onTabChange={onTabChange}
                tabs={[
                    {
                        id: TAB_TYPE.LIST,
                        label: intl.getMessage('blocklist_add_from_list'),
                        content: <FiltersList selectedSources={selectedSources} />,
                    },
                    {
                        id: TAB_TYPE.MANUAL,
                        label: intl.getMessage('blocklist_add_manual'),
                        content: <ManualFilterForm />,
                    },
                ]}
            />
        );
    }

    if (modalType === MODAL_TYPE.CHOOSE_FILTERING_LIST) {
        return <FiltersList selectedSources={selectedSources} />;
    }

    return <ManualFilterForm />;
};
