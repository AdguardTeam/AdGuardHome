import React, { forwardRef, useState } from 'react';

import { Input } from 'panel/common/controls/Input/index';
import { Icon } from 'panel/common/ui/Icon';

import styles from './Input.module.pcss';

type Props = Omit<React.ComponentProps<typeof Input>, 'type' | 'suffixIcon' | 'onChange' | 'value'> & {
    value: string;
    onChange: (value: string) => void;
};

export const PasswordInput = forwardRef<HTMLInputElement, Props>(({ value, onChange, ...props }, ref) => {
    const [isPasswordVisible, setIsPasswordVisible] = useState(false);

    return (
        <Input
            {...props}
            ref={ref}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            type={isPasswordVisible ? 'text' : 'password'}
            suffixIcon={(
                <div className={styles.inputSuffix}>
                    {!!value && (
                        <button
                            className={styles.inputIconButton}
                            tabIndex={-1}
                            type="button"
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => {
                                setIsPasswordVisible(false);
                                onChange('');
                            }}
                        >
                            <Icon icon="cross" />
                        </button>
                    )}
                    <button
                        className={styles.inputIconButton}
                        tabIndex={-1}
                        type="button"
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => setIsPasswordVisible((v) => !v)}
                    >
                        <Icon icon={isPasswordVisible ? 'eye_open' : 'eye_close'} />
                    </button>
                </div>
            )}
        />
    );
});

PasswordInput.displayName = 'PasswordInput';
