import React from 'react';

type Props = {
    id?: string;
    className?: string;
    placeholder?: string;
    type?: string;
    disabled?: boolean;
    autoComplete?: string;
    isActionAvailable?: boolean;
    removeField?: () => void;
    normalizeOnBlur?: (value: string) => string;
    value: string;
    onChange: (value: string) => void;
    onBlur: () => void;
};

export const InputGroup = ({
    id,
    className,
    placeholder,
    type,
    disabled,
    autoComplete,
    isActionAvailable,
    removeField,
    normalizeOnBlur,
    value,
    onChange,
    onBlur,
}: Props) => {
    const handleBlur = (event: React.FocusEvent<HTMLInputElement>) => {
        if (normalizeOnBlur) {
            onChange(normalizeOnBlur(event.target.value));
        }
        onBlur();
    };

    return (
        <div className="input-group">
            <input
                id={id}
                placeholder={placeholder}
                type={type}
                className={className}
                disabled={disabled}
                autoComplete={autoComplete}
                value={value}
                onChange={(e) => onChange(e.target.value)}
                onBlur={handleBlur}
            />
            {isActionAvailable && (
                <span className="input-group-append">
                    <button type="button" className="btn btn-secondary btn-icon btn-icon--green" onClick={removeField}>
                        <svg className="icon icon--24">
                            <use xlinkHref="#cross" />
                        </svg>
                    </button>
                </span>
            )}
        </div>
    );
};
