import { createSignal, Show } from 'solid-js';

import { Input } from 'panel/common/controls/Input/index';
import { Icon } from 'panel/common/ui/Icon';

import styles from './Input.module.pcss';

type Props = Omit<Parameters<typeof Input>[0], 'type' | 'suffixIcon' | 'onChange' | 'value'> & {
    value: string;
    onChange: (value: string) => void;
    ref?: HTMLInputElement | ((el: HTMLInputElement) => void);
};

export const PasswordInput = (props: Props) => {
    const [isPasswordVisible, setIsPasswordVisible] = createSignal(false);

    return (
        <Input
            {...props}
            ref={props.ref}
            value={props.value}
            onChange={(e) => props.onChange(e.target.value)}
            type={isPasswordVisible() ? 'text' : 'password'}
            suffixIcon={
                <div class={styles.inputSuffix}>
                    <Show when={!!props.value}>
                        <button
                            class={styles.inputIconButton}
                            tabIndex={-1}
                            type="button"
                            onMouseDown={(e: MouseEvent) => e.preventDefault()}
                            onClick={() => {
                                setIsPasswordVisible(false);
                                props.onChange('');
                            }}
                        >
                            <Icon icon="cross" />
                        </button>
                    </Show>
                    <button
                        class={styles.inputIconButton}
                        tabIndex={-1}
                        type="button"
                        onMouseDown={(e: MouseEvent) => e.preventDefault()}
                        onClick={() => setIsPasswordVisible((v) => !v)}
                    >
                        <Icon icon={(isPasswordVisible() ? 'eye_open' : 'eye_close') as any} />
                    </button>
                </div>
            }
        />
    );
};
