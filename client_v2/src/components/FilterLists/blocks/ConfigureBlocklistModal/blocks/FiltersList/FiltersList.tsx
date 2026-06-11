import React from 'react';
import { Controller, useFormContext } from 'react-hook-form';
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
};

export const FiltersList = ({ selectedSources }: Props) => {
    const { control } = useFormContext();
    const { categories, filters } = filtersCatalog;

    return (
        <div className={s.listWrap}>
            <div className={s.list}>
                {Object.entries(categories).map(([categoryId, _category]) => {
                    const categoryFilters = Object.entries(filters)
                        .filter(([, filter]) => filter.categoryId === categoryId)
                        .map(([key, filter]) => ({ ...filter, id: key }));

                    return (
                        <div key={categoryId}>
                            <div className={s.category}>
                                <div className={cn(theme.text.t2, s.categoryName)}>
                                    {getCategoryName(categoryId)}
                                </div>
                                <div className={cn(theme.text.t3, s.categoryDescription)}>
                                    {getCategoryDesc(categoryId)}
                                </div>
                            </div>

                            {categoryFilters.map((filter) => {
                                const { homepage, source, name, id } = filter;
                                const isSelected = selectedSources?.[source] || false;

                                return (
                                    <div key={name} className={s.filter}>
                                        <Controller
                                            name={id}
                                            control={control}
                                            render={({ field: { value, onChange, ...field } }) => (
                                                <Checkbox
                                                    {...field}
                                                    checked={!!value}
                                                    onChange={onChange}
                                                    id={`filters_${id}`}
                                                    title={name}
                                                    disabled={isSelected}
                                                >
                                                    {name}
                                                </Checkbox>
                                            )}
                                        />
                                        <Dropdown
                                            trigger="click"
                                            menu={
                                                <div className={theme.dropdown.menu}>
                                                    <a
                                                        href={homepage}
                                                        className={theme.dropdown.item}
                                                        target="_blank"
                                                        rel="noreferrer"
                                                    >
                                                        <Icon icon="link" color="green" />
                                                        {intl.getMessage('blocklist_homepage')}
                                                    </a>
                                                    <a
                                                        href={source}
                                                        className={theme.dropdown.item}
                                                        target="_blank"
                                                        rel="noreferrer"
                                                    >
                                                        <Icon icon="link" color="green" />
                                                        {intl.getMessage('blocklist_contents', {
                                                            value: 'txt',
                                                        })}
                                                    </a>
                                                </div>
                                            }
                                            position="bottomRight"
                                            noIcon
                                        >
                                            <div className={s.dropdownTrigger}>
                                                <Icon icon="bullets" color="gray" />
                                            </div>
                                        </Dropdown>
                                    </div>
                                );
                            })}
                        </div>
                    );
                })}
            </div>
        </div>
    );
};
