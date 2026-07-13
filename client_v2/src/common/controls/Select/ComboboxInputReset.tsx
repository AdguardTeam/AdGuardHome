import { type JSX, createEffect, on } from 'solid-js';
import { useComboboxContext } from '@ark-ui/solid';

export const ComboboxInputReset = (): JSX.Element => {
    const comboCtx = useComboboxContext();

    createEffect(
        on(
            () => comboCtx().open,
            (nowOpen) => {
                if (!nowOpen) {
                    comboCtx().setInputValue('');
                }
            },
            { defer: true },
        ),
    );

    return null;
};
