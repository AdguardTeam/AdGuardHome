import React from 'react';
import { components, MultiValueProps } from 'react-select';
import intl from 'panel/common/intl';

export const CustomMultiValue = <T extends Record<string, any> = any>(props: MultiValueProps<T, true>) => {
    const { getValue } = props;
    const selectedValues = getValue();
    const selectedCount = selectedValues.length;

    if (props.index === 0) {
        return (
            <components.MultiValue
                {...props}
                cropWithEllipsis={false}
                innerProps={{
                    ...props.innerProps,
                }}>
                {selectedCount === 1 ? selectedValues[0]?.label : intl.getMessage('selected', { value: selectedCount })}
            </components.MultiValue>
        );
    }

    return null;
};
