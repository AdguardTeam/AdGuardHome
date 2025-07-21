import React from 'react';

interface CellWrapProps {
    value?: string | number;
    formatValue?: (...args: unknown[]) => unknown;
    formatTitle?: (...args: unknown[]) => unknown;
}

const CellWrap = ({ value }: CellWrapProps, formatValue?: any, formatTitle = formatValue) => {
    if (!value) {
        return '–';
    }
    const cellValue = typeof formatValue === 'function' ? formatValue(value) : value;
    const cellTitle = typeof formatTitle === 'function' ? formatTitle(value) : value;

    return (
        <div className="logs__row o-hidden">
            <span className="logs__text logs__text--full" title={cellTitle}>
                {cellValue}
            </span>
        </div>
    );
};

export default CellWrap;
