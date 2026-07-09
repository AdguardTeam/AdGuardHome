import { For, untrack } from 'solid-js';
import cn from 'clsx';

import { Checkbox } from 'panel/common/controls/Checkbox';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import filtersCatalog from 'panel/helpers/filters/filters';

import s from './FiltersList.module.pcss';

const getCategoryName = (categoryId: string) => {
    switch (categoryId) {
        case 'general':
            return intl.getMessage('filter_category_general');
        case 'security':
            return intl.getMessage('filter_category_security');
        case 'regional':
            return intl.getMessage('filter_category_regional');
        case 'other':
            return intl.getMessage('filter_category_other');
        default:
            return categoryId;
    }
};

const getCategoryDesc = (categoryId: string) => {
    switch (categoryId) {
        case 'general':
            return intl.getMessage('filter_category_general_desc');
        case 'security':
            return intl.getMessage('filter_category_security_desc');
        case 'regional':
            return intl.getMessage('filter_category_regional_desc');
        case 'other':
            return intl.getMessage('filter_category_other_desc');
        default:
            return categoryId;
    }
};

type Props = {
    selectedSources?: Record<string, boolean>;
    selectedIds: Record<string, boolean>;
    onChange: (ids: Record<string, boolean>) => void;
    disabled?: boolean;
};

export const FiltersList = (props: Props) => {
    const { categories, filters } = filtersCatalog;

    const handleToggle = (filterId: string) => (e: Event) => {
        untrack(() => {
            const checked = (e.target as HTMLInputElement).checked;
            props.onChange({
                ...props.selectedIds,
                [filterId]: checked,
            });
        });
    };

    return (
        <div class={s.listWrap}>
            <div class={s.list}>
                <For each={Object.entries(categories)}>
                    {([categoryId]) => {
                        const categoryFilters = Object.entries(filters)
                            .filter(([, filter]) => filter.categoryId === categoryId)
                            .map(([key, filter]) => ({ ...filter, id: key }));

                        return (
                            <div>
                                <div class={s.category}>
                                    <div class={cn(theme.text.t2, s.categoryName)}>
                                        {getCategoryName(categoryId)}
                                    </div>
                                    <div class={cn(theme.text.t3, s.categoryDescription)}>
                                        {getCategoryDesc(categoryId)}
                                    </div>
                                </div>

                                <For each={categoryFilters}>
                                    {(filter) => {
                                        const { homepage, source, name, id } = filter;
                                        return (
                                            <div class={s.filter}>
                                                <Checkbox
                                                    checked={!!props.selectedIds[id]}
                                                    onChange={handleToggle(id)}
                                                    id={`filters_${id}`}
                                                    title={name}
                                                    disabled={props.disabled}
                                                >
                                                    {name}
                                                </Checkbox>
                                                <Dropdown
                                                    trigger="click"
                                                    menu={
                                                        <div class={theme.dropdown.menu}>
                                                            <a
                                                                href={homepage}
                                                                class={theme.dropdown.item}
                                                                target="_blank"
                                                                rel="noreferrer"
                                                            >
                                                                <Icon icon="link" color="green" />
                                                                {intl.getMessage(
                                                                    'blocklist_homepage',
                                                                )}
                                                            </a>
                                                            <a
                                                                href={source}
                                                                class={theme.dropdown.item}
                                                                target="_blank"
                                                                rel="noreferrer"
                                                            >
                                                                <Icon icon="link" color="green" />
                                                                {intl.getMessage(
                                                                    'blocklist_contents',
                                                                    {
                                                                        value: 'txt',
                                                                    },
                                                                )}
                                                            </a>
                                                        </div>
                                                    }
                                                    position="bottomRight"
                                                    noIcon
                                                >
                                                    <div class={s.dropdownTrigger}>
                                                        <Icon icon="bullets" color="gray" />
                                                    </div>
                                                </Dropdown>
                                            </div>
                                        );
                                    }}
                                </For>
                            </div>
                        );
                    }}
                </For>
            </div>
        </div>
    );
};
