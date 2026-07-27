import { For } from 'solid-js';

import { Toast } from './Toast';
import './Toast.pcss';
import { toastsState } from 'panel/stores/toasts';

export const Toasts = () => {
    return (
        <div class="toasts">
            <For each={toastsState.notices}>{(toast: any) => <Toast {...toast} />}</For>
        </div>
    );
};
