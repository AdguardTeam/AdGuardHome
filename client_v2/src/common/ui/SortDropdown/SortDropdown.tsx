import React, { useState } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import theme from 'panel/lib/theme';

type Props = {
    value: 'asc' | 'desc';
    onChange: (value: 'asc' | 'desc') => void;
};

export const SortDropdown = ({ value, onChange }: Props) => {
    const [open, setOpen] = useState(false);

    const menu = (
        <div className={theme.dropdown.menu}>
            <div
                key="asc"
                className={cn(theme.dropdown.item, {
                    [theme.dropdown.item_active]: value === 'asc',
                })}
                onClick={() => {
                    onChange('asc');
                    setOpen(false);
                }}
            >
                {intl.getMessage('sort_asc')}
            </div>

            <div
                key="desc"
                className={cn(theme.dropdown.item, {
                    [theme.dropdown.item_active]: value === 'desc',
                })}
                onClick={() => {
                    onChange('desc');
                    setOpen(false);
                }}
            >
                {intl.getMessage('sort_desc')}
            </div>
        </div>
    );

    return (
        <Dropdown
            trigger="click"
            position="bottomLeft"
            menu={menu}
            open={open}
            onOpenChange={setOpen}
            iconClassName={theme.dropdown.icon}
            className={theme.dropdown.flexDropdownWrap}
            wrapClassName={cn(theme.dropdown.dropdown, theme.pagination.dropdownShowOnPage)}
        >
            <span className={theme.pagination.dropdownText}>
                {intl.getMessage(value === 'asc' ? 'sort_asc' : 'sort_desc')}
            </span>
        </Dropdown>
    );
};
