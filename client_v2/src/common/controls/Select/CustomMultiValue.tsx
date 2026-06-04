import React from 'react';
import { components, MultiValueProps } from 'react-select';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';

import s from './CustomMultiValue.module.pcss';

export const CustomMultiValue = <T extends Record<string, any> = any>(
    props: MultiValueProps<T, true>,
) => {
    const { data, removeProps } = props;
    const label = data.label as string;

    return (
        <components.MultiValue {...props}>
            <div className={s.pill}>
                <span className={s.label}>{label}</span>
                <button
                    type="button"
                    className={s.removeBtn}
                    onClick={(e) =>
                        removeProps.onClick?.(e as unknown as React.MouseEvent<HTMLDivElement>)
                    }
                    onMouseDown={(e) =>
                        removeProps.onMouseDown?.(e as unknown as React.MouseEvent<HTMLDivElement>)
                    }
                    onTouchEnd={(e) =>
                        removeProps.onTouchEnd?.(e as unknown as React.TouchEvent<HTMLDivElement>)
                    }
                    aria-label={intl.getMessage('remove_tag', { value: label })}
                >
                    <Icon icon="cross" color="gray" />
                </button>
            </div>
        </components.MultiValue>
    );
};
