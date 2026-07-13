import { createSignal, createEffect, onCleanup } from 'solid-js';

const useDebounce = (value: any, delay: any) => {
    const [debouncedValue, setDebouncedValue] = createSignal(value);

    createEffect(() => {
        const handler = setTimeout(() => {
            setDebouncedValue(value);
        }, delay);

        onCleanup(() => {
            clearTimeout(handler);
        });
    });

    return [debouncedValue, setDebouncedValue];
};

export default useDebounce;
