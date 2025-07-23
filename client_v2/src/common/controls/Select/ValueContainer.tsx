import React from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Loader } from 'panel/common/ui/Loader';
import { IOption } from 'panel/lib/helpers/utils';
import theme from 'panel/lib/theme';
import { ISelectValue } from './Select';

type Props<T, Multi extends boolean = true> = {
    value?: ISelectValue<T, Multi>;
    onClear: () => void;
    onClose: () => void;
    isMulti: Multi;
    loading: boolean;
    placeholder: string;
    isClearable?: boolean;
};

export const ValueContainer = <T, Multi extends boolean = true>({
    value,
    onClear,
    onClose,
    placeholder,
    isMulti,
    loading,
    isClearable,
}: Props<T, Multi>) => {
    const isEmpty = !value || (Array.isArray(value) && value.length === 0);

    const renderValue = () => {
        if (isEmpty) {
            return placeholder;
        }

        if (isMulti) {
            return Array.isArray(value) && value.length === 1 && value[0]
                ? value[0].label
                : intl.getMessage('selected', {
                      value: Array.isArray(value) ? value.length : 0,
                  });
        }

        if (Array.isArray(value)) {
            return value[0].label;
        }

        return (value as IOption<T>).label;
    };

    const renderLoader = () =>
        loading ? (
            <Loader overlayClassName={theme.select.loaderOverlay} className={theme.select.loader} icon="loader" />
        ) : null;

    const renderClear = () => {
        if (isEmpty || !isClearable) {
            return null;
        }

        return (
            <div
                className={theme.select.clearIndicator}
                onClick={(e) => {
                    e.stopPropagation();
                    onClear();
                    onClose();
                }}>
                <Icon icon="cross" className={theme.select.clearIcon} />
            </div>
        );
    };

    return (
        <>
            <span className={cn(theme.common.textOverflow, theme.select.dropdownText)}>{renderValue()}</span>
            {renderLoader()}
            {renderClear()}
        </>
    );
};
