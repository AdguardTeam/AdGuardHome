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

type Props = {
    selectedSources?: Record<string, boolean>;
};

export const FiltersList = ({ selectedSources }: Props) => {
    const { control } = useFormContext();
    const { categories, filters } = filtersCatalog;

    return (
        <div className={s.listWrap}>
            <div className={s.list}>
                {Object.entries(categories).map(([categoryId, category]) => {
                    const categoryFilters = Object.entries(filters)
                        .filter(([, filter]) => filter.categoryId === categoryId)
                        .map(([key, filter]) => ({ ...filter, id: key }));

                    return (
                        <div key={category.name}>
                            <div className={s.category}>
                                <div className={cn(theme.text.t2, s.categoryName)}>
                                    {intl.getMessage(category.name)}
                                </div>
                                <div className={cn(theme.text.t3, s.categoryDescription)}>
                                    {intl.getMessage(category.description)}
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
                                                    disabled={isSelected}>
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
                                                        rel="noreferrer">
                                                        <Icon icon="link" color="green" />
                                                        {intl.getMessage('blocklist_homepage')}
                                                    </a>
                                                    <a
                                                        href={source}
                                                        className={theme.dropdown.item}
                                                        target="_blank"
                                                        rel="noreferrer">
                                                        <Icon icon="link" color="green" />
                                                        {intl.getMessage('blocklist_contents', { value: 'txt' })}
                                                    </a>
                                                </div>
                                            }
                                            position="bottomRight"
                                            noIcon>
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
