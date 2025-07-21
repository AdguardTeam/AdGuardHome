import React from 'react';
import { components, ValueContainerProps } from 'react-select';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

export const CustomValueContainer = <T extends Record<string, any> = any, IsMulti extends boolean = true>(
    props: ValueContainerProps<T, IsMulti>,
) => {
    const { hasValue, children, getValue } = props;

    if (!hasValue) {
        return (
            <components.ValueContainer {...props} hasValue={hasValue}>
                {props.children}
            </components.ValueContainer>
        );
    }

    const selectedValues = getValue();
    const selectedCount = selectedValues.length;

    return (
        <components.ValueContainer {...props}>
            <div className={theme.common.textOverflow}>
                {selectedCount === 1 ? selectedValues[0]?.label : intl.getMessage('selected', { value: selectedCount })}
            </div>
            <div className={theme.layout.visuallyHidden}>{children}</div>
        </components.ValueContainer>
    );
};
