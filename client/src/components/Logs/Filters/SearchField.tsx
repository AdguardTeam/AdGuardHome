import React, { ComponentProps } from 'react';
import Tooltip from '../../ui/Tooltip';

interface Props extends ComponentProps<'input'> {
    handleChange: (newValue: string) => void;
    onClear: () => void;
    tooltip?: string;
}

export const SearchField = ({
    handleChange,
    onClear,
    value,
    tooltip,
    className,
    ...rest
}: Props) => {
    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        handleChange(e.target.value);
    };

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
        e.target.value = e.target.value.trim();
        handleChange(e.target.value)
    }

    return (
        <>
            <div className="input-group-search input-group-search__icon--magnifier">
                <svg className="icons icon--24 icon--gray">
                    <use xlinkHref="#magnifier" />
                </svg>
            </div>
            <input
                className={className}
                value={value}
                onChange={handleInputChange}
                onBlur={handleBlur}
                {...rest}
            />
            {typeof value === 'string' && value.length > 0 && (
                <div
                    className="input-group-search input-group-search__icon--cross"
                    onClick={onClear}
                >
                    <svg className="icons icon--20 icon--gray">
                    <use xlinkHref="#cross" />
                    </svg>
                </div>
            )}
            {tooltip && (
                <span className="input-group-search input-group-search__icon--tooltip">
                    <Tooltip content={tooltip} className="tooltip-container">
                        <svg className="icons icon--20 icon--gray">
                            <use xlinkHref="#question" />
                        </svg>
                    </Tooltip>
                </span>
            )}
        </>
    );
};
