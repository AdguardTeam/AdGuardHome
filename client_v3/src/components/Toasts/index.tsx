import { For } from 'solid-js';

import Toast from './Toast';
import './Toast.pcss';
import { toastsState } from 'panel/stores/toasts';

const Toasts = () => {
    return (
        <div class="toasts">
            <For each={toastsState.notices}>{(toast: any) => <Toast {...toast} />}</For>
        </div>
    );
};

export default Toasts;
