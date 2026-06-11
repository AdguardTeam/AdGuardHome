import React, { useMemo } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Select } from 'panel/common/controls/Select';
import { IOption } from 'panel/lib/helpers/utils';
import s from './SortSelect.module.pcss';

type Props = {
    value: 'asc' | 'desc';
    onChange: (value: 'asc' | 'desc') => void;
    className?: string;
};

export const SortSelect = ({ value, onChange, className }: Props) => {
    const options: IOption<string>[] = useMemo(
        () => [
            { value: 'asc', label: intl.getMessage('sort_asc') },
            { value: 'desc', label: intl.getMessage('sort_desc') },
        ],
        [],
    );

    return (
        <div className={cn(s.wrapper, className)}>
            <Select<string>
                options={options}
                value={options.find((o) => o.value === value)}
                onChange={(option) => onChange(option.value as 'asc' | 'desc')}
                height="medium"
                isSearchable={false}
            />
        </div>
    );
};
