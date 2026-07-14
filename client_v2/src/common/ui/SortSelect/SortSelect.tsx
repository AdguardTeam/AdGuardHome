import cn from 'clsx';

import intl from 'panel/common/intl';
import { Select } from 'panel/common/controls/Select';
import { type IOption } from 'panel/lib/helpers/utils';
import s from './SortSelect.module.pcss';

type Props = {
    value: 'asc' | 'desc';
    onChange: (value: 'asc' | 'desc') => void;
    class?: string;
};

export const SortSelect = (props: Props) => {
    const options: IOption<string>[] = [
        { value: 'asc', label: intl.getMessage('sort_asc') },
        { value: 'desc', label: intl.getMessage('sort_desc') },
    ];

    return (
        <div class={cn(s.wrapper, props.class)}>
            <Select<string>
                options={options}
                value={options.find((o) => o.value === props.value)}
                onChange={(option: any) => props.onChange(option.value as 'asc' | 'desc')}
                height="medium"
                isSearchable={false}
            />
        </div>
    );
};
